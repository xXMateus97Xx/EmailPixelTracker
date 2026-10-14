package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

// captureLogs sends the default logger to a buffer, at debug level, for the
// duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	logger, err := telemetry.NewLogger(&buf, "debug", "json")
	if err != nil {
		t.Fatal(err)
	}
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		lines = append(lines, m)
	}
	return lines
}

func findLog(lines []map[string]any, msg string) map[string]any {
	for _, line := range lines {
		if line["msg"] == msg {
			return line
		}
	}
	return nil
}

func serve(router *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRequestId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, err := newDocsRouter(t, RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	generated := get(router, "/health/live").Header().Get(requestIdHeader)
	if len(generated) != 36 {
		t.Errorf("generated request id %q", generated)
	}

	for header, want := range map[string]string{
		"from-proxy.01:a_b":     "from-proxy.01:a_b",
		"has spaces":            "",
		"x\nforged=1":           "",
		strings.Repeat("a", 65): "",
	} {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		req.Header.Set(requestIdHeader, header)
		got := serve(router, req).Header().Get(requestIdHeader)
		if want != "" && got != want {
			t.Errorf("%q: request id %q, want it kept", header, got)
		}
		if want == "" && (got == header || len(got) != 36) {
			t.Errorf("%q: request id %q, want a generated one", header, got)
		}
	}
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := captureLogs(t)

	var dbErr error
	router, err := newDocsRouter(t, RouterConfig{Readiness: func(context.Context) error { return dbErr }})
	if err != nil {
		t.Fatal(err)
	}

	if rec := get(router, "/health/ready"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Errorf("ready: status %d, body %s", rec.Code, rec.Body.String())
	}

	dbErr = errors.New("dial tcp 10.0.0.5:5432: connection refused")
	rec := get(router, "/health/ready")
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != `{"status":"unavailable"}` {
		t.Errorf("not ready: status %d, body %s", rec.Code, rec.Body.String())
	}
	if line := findLog(logLines(t, logs), "readiness check failed"); line == nil || line["error"] != dbErr.Error() {
		t.Errorf("readiness failure not logged: %s", logs.String())
	}

	// Liveness does not depend on the database.
	if rec := get(router, "/health/live"); rec.Code != http.StatusOK {
		t.Errorf("live: status %d", rec.Code)
	}
}

func TestPanicRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := captureLogs(t)
	router, err := newDocsRouter(t, RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/boom", func(*gin.Context) { panic("boom") })

	rec := get(router, "/boom")
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError || body.RequestId == "" || body.RequestId != rec.Header().Get(requestIdHeader) {
		t.Fatalf("status %d, body %s, header %q", rec.Code, rec.Body.String(), rec.Header().Get(requestIdHeader))
	}

	lines := logLines(t, logs)
	panicLog := findLog(lines, "panic while serving request")
	if panicLog == nil || panicLog["request_id"] != body.RequestId || !strings.Contains(panicLog["stack"].(string), "observability") {
		t.Errorf("panic not logged with request id and stack: %s", logs.String())
	}
	if line := findLog(lines, "http request"); line == nil || line["level"] != "ERROR" || line["status"] != float64(500) {
		t.Errorf("request log %v", line)
	}
}

func TestRequestLogsAndMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := captureLogs(t)
	metrics := telemetry.NewMetrics()
	router, err := newDocsRouter(t, RouterConfig{Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}

	get(router, "/pixel/unknown?externalUserId=customer-secret-42")
	get(router, "/api/campaigns")
	get(router, "/random/scan/path")
	serve(router, httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"username":"nobody","password":"my-password"}`)))

	out := logs.String()
	for _, secret := range []string{"customer-secret-42", "my-password", "nobody", "/random/scan/path"} {
		if strings.Contains(out, secret) {
			t.Errorf("logs contain %q: %s", secret, out)
		}
	}

	lines := logLines(t, logs)
	var pixelLog, unauthorizedLog map[string]any
	for _, line := range lines {
		if line["msg"] != "http request" {
			continue
		}
		switch line["route"] {
		case "/pixel/:campaignId":
			pixelLog = line
		case "/api/campaigns":
			unauthorizedLog = line
		}
	}
	if pixelLog == nil || pixelLog["level"] != "DEBUG" || pixelLog["status"] != float64(404) {
		t.Errorf("pixel request log %v", pixelLog)
	}
	if unauthorizedLog == nil || unauthorizedLog["level"] != "INFO" || unauthorizedLog["request_id"] == nil {
		t.Errorf("request log %v", unauthorizedLog)
	}
	if line := findLog(lines, "pixel view not tracked"); line == nil || line["reason"] != "campaign_not_found" {
		t.Errorf("pixel reason not logged: %v", line)
	}
	if line := findLog(lines, "login failed"); line == nil || line["reason"] != "invalid_credentials" {
		t.Errorf("login failure not logged: %v", line)
	}

	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, want := range []string{
		`email_tracker_pixel_requests_total{result="campaign_not_found"} 1`,
		`email_tracker_http_requests_total{method="GET",route="/pixel/:campaignId",status="404"} 1`,
		`email_tracker_http_requests_total{method="GET",route="/api/campaigns",status="401"} 1`,
		`email_tracker_http_requests_total{method="GET",route="unmatched",status="404"} 1`,
		`email_tracker_logins_total{result="invalid_credentials"} 1`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("metrics do not contain %s", want)
		}
	}
}

// Authenticated requests are logged with the user and the effective tenant.
func TestRequestLogsPrincipal(t *testing.T) {
	c := newClient(t)
	root := c.login("root", "rootpassword")
	var tenant tenantResponse
	c.do(http.MethodPost, "/api/tenants", root, map[string]any{"name": "Acme"}, http.StatusCreated, &tenant)

	logs := captureLogs(t)
	var imp tokenResponse
	c.do(http.MethodPost, "/api/auth/impersonate", root, impersonateRequest{TenantId: tenant.Id}, http.StatusOK, &imp)
	c.do(http.MethodGet, "/api/campaigns", imp.Token, nil, http.StatusOK, nil)

	lines := logLines(t, logs)
	if line := findLog(lines, "impersonation started"); line == nil || line["impersonated_tenant_id"] != tenant.Id {
		t.Errorf("impersonation not logged: %v", line)
	}
	var found bool
	for _, line := range lines {
		if line["msg"] == "http request" && line["route"] == "/api/campaigns" {
			found = true
			if line["tenant_id"] != tenant.Id || line["role"] != "SuperAdmin" || line["impersonating"] != true || line["user_id"] == nil {
				t.Errorf("request log %v", line)
			}
		}
	}
	if !found {
		t.Errorf("request not logged: %s", logs.String())
	}
}
