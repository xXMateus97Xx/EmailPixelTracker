package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestLoggerAddsRequestAndTraceIds(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, "info", "json")
	if err != nil {
		t.Fatal(err)
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3},
		SpanID:     trace.SpanID{4, 5, 6},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(WithRequestId(context.Background(), "req-1"), sc)

	logger.InfoContext(ctx, "with context")
	logger.With("component", "test").InfoContext(ctx, "derived logger")
	logger.Info("without context")
	logger.DebugContext(ctx, "below the level")

	lines := decodeLines(t, &buf)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %s", len(lines), buf.String())
	}
	for _, line := range lines[:2] {
		if line["request_id"] != "req-1" || line["trace_id"] != sc.TraceID().String() || line["span_id"] != sc.SpanID().String() {
			t.Errorf("missing ids: %v", line)
		}
	}
	if _, ok := lines[2]["request_id"]; ok {
		t.Errorf("ids without context: %v", lines[2])
	}
}

func TestLoggerConfig(t *testing.T) {
	for _, tt := range []struct{ level, format string }{{"verbose", "json"}, {"info", "xml"}} {
		if _, err := NewLogger(&bytes.Buffer{}, tt.level, tt.format); err == nil {
			t.Errorf("level %q, format %q: expected an error", tt.level, tt.format)
		}
	}
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, "WARN", "text")
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hidden")
	logger.Warn("shown")
	if out := buf.String(); strings.Contains(out, "hidden") || !strings.Contains(out, "level=WARN msg=shown") {
		t.Errorf("unexpected output %q", out)
	}
}

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	return rec.Body.String()
}

func TestMetrics(t *testing.T) {
	var none *Metrics
	none.ObserveHTTP("/x", "GET", 200, time.Millisecond)
	none.PixelRequest("tracked")
	none.PixelCached()
	none.Login("success")
	none.Impersonation("start")
	none.ObserveDB("postgres", "query", time.Millisecond, false)
	if err := none.RegisterSQLStats(nil, "x"); err != nil {
		t.Fatal(err)
	}

	m := NewMetrics()
	m.ObserveHTTP("/pixel/:campaignId", "GET", 404, 2*time.Millisecond)
	m.PixelRequest("campaign_expired")
	m.Login("invalid_credentials")
	m.Impersonation("start")
	m.ObserveDB("postgres", "query", time.Millisecond, true)

	out := scrape(t, m)
	for _, want := range []string{
		`email_tracker_http_requests_total{method="GET",route="/pixel/:campaignId",status="404"} 1`,
		`email_tracker_http_request_duration_seconds_count{method="GET",route="/pixel/:campaignId"} 1`,
		`email_tracker_pixel_requests_total{result="campaign_expired"} 1`,
		`email_tracker_logins_total{result="invalid_credentials"} 1`,
		`email_tracker_impersonations_total{action="start"} 1`,
		`email_tracker_db_operation_duration_seconds_count{operation="query",status="error",system="postgres"} 1`,
		`go_goroutines`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics do not contain %s", want)
		}
	}
}

func TestMongoMonitor(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	m := NewMetrics()
	monitor := MongoMonitor(m)
	command, err := bson.Marshal(bson.D{{Key: "find", Value: "campaigns"}, {Key: "filter", Value: bson.D{{Key: "secret", Value: "x"}}}})
	if err != nil {
		t.Fatal(err)
	}
	run := func(ctx context.Context, requestId int64, failure error) {
		monitor.Started(ctx, &event.CommandStartedEvent{
			Command: command, DatabaseName: "db", CommandName: "find", RequestID: requestId, ConnectionID: "conn",
		})
		finished := event.CommandFinishedEvent{CommandName: "find", DatabaseName: "db", RequestID: requestId, ConnectionID: "conn", Duration: time.Millisecond}
		if failure != nil {
			monitor.Failed(ctx, &event.CommandFailedEvent{CommandFinishedEvent: finished, Failure: failure})
		} else {
			monitor.Succeeded(ctx, &event.CommandSucceededEvent{CommandFinishedEvent: finished})
		}
	}

	// Outside a trace (startup, health checks): measured, not traced.
	run(context.Background(), 1, nil)
	if n := len(recorder.Ended()); n != 0 {
		t.Fatalf("%d spans without a parent, want 0", n)
	}

	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	run(ctx, 2, nil)
	run(ctx, 3, errors.New("boom"))
	parent.End()

	var spans []sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "find campaigns" {
			spans = append(spans, s)
		}
	}
	if len(spans) != 2 {
		t.Fatalf("got %d command spans, want 2", len(spans))
	}
	if spans[0].Status().Code == codes.Error || spans[1].Status().Code != codes.Error {
		t.Errorf("statuses %v, %v", spans[0].Status(), spans[1].Status())
	}
	for _, kv := range spans[0].Attributes() {
		if strings.Contains(kv.Value.Emit(), "secret") {
			t.Errorf("span leaks the command body: %v", kv)
		}
	}

	out := scrape(t, m)
	for _, want := range []string{
		`email_tracker_db_operation_duration_seconds_count{operation="find",status="ok",system="mongodb"} 2`,
		`email_tracker_db_operation_duration_seconds_count{operation="find",status="error",system="mongodb"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics do not contain %s", want)
		}
	}
}
