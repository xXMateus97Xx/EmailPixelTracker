package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ServiceName identifies the application in traces; OTEL_SERVICE_NAME overrides it.
const ServiceName = "email-pixel-tracker"

const instrumentationName = "github.com/xXMateus97Xx/email-pixel-tracker/telemetry"

// SetupTracing exports traces with OTLP over HTTP when
// OTEL_EXPORTER_OTLP_ENDPOINT (or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT) is set;
// otherwise tracing stays disabled and the instrumentation costs next to
// nothing. The standard OTEL_* variables (headers, sampler, resource
// attributes) are honored. The returned function flushes pending spans.
func SetupTracing(ctx context.Context, version string) (enabled bool, shutdown func(context.Context) error, err error) {
	// Incoming W3C trace context is continued even when tracing is disabled,
	// so the ids still reach the logs of upstream systems.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Warn("opentelemetry error", "error", err)
	}))

	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return false, noop, nil
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return false, noop, err
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", ServiceName),
			attribute.String("service.version", version),
		),
		// Applied last, so OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES win.
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return false, noop, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return true, provider.Shutdown, nil
}
