// Package telemetry holds the observability building blocks: structured logs,
// Prometheus metrics, OpenTelemetry tracing and the database instrumentation.
package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// NewLogger builds the application logger. level is debug, info, warn or
// error (default info); format is json or text (default json).
// Records logged with a context carry its request_id, trace_id and span_id.
func NewLogger(w io.Writer, level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if level != "" {
		if err := lvl.UnmarshalText([]byte(level)); err != nil {
			return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error: %w", err)
		}
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "", "json":
		handler = slog.NewJSONHandler(w, opts)
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be json or text, got %q", format)
	}
	return slog.New(contextHandler{handler}), nil
}

// contextHandler adds the request and trace identifiers found in the context,
// so every log of a request can be correlated with the others and its trace.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestId(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

type requestIdKey struct{}

func WithRequestId(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIdKey{}, id)
}

// RequestId returns the id of the request being served, or "".
func RequestId(ctx context.Context) string {
	id, _ := ctx.Value(requestIdKey{}).(string)
	return id
}
