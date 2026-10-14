package api

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xXMateus97Xx/email-pixel-tracker/auth"
	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

const (
	requestIdHeader = "X-Request-ID"
	readyTimeout    = 3 * time.Second
	// unmatchedRoute labels requests that match no route, so scans of random
	// paths do not create a metric series per path.
	unmatchedRoute = "unmatched"
)

// validRequestId accepts ids from a proxy or client as long as they are short
// and cannot inject anything into the logs.
var validRequestId = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

// observability returns the middlewares, outermost first: request id,
// tracing, request log and metrics, panic recovery.
func observability(metrics *telemetry.Metrics) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		requestId(),
		otelgin.Middleware(telemetry.ServiceName, otelgin.WithGinFilter(func(c *gin.Context) bool {
			return !strings.HasPrefix(c.FullPath(), "/health/")
		})),
		observe(metrics),
		recoverPanic(),
	}
}

// requestId reuses a valid X-Request-ID received or generates one, returns it
// in the response and puts it in the context for the logs.
func requestId() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIdHeader)
		if !validRequestId.MatchString(id) {
			id = uuid.NewString()
		}
		c.Header(requestIdHeader, id)
		c.Request = c.Request.WithContext(telemetry.WithRequestId(c.Request.Context(), id))
		c.Next()
	}
}

// observe logs one line per request and records its metrics. The route
// pattern is used instead of the path, and the query string is never logged:
// it carries the externalUserId of the pixel.
func observe(metrics *telemetry.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		route := c.FullPath()
		if route == "" {
			route = unmatchedRoute
		}
		status := c.Writer.Status()
		metrics.ObserveHTTP(route, c.Request.Method, status, duration)

		ctx := c.Request.Context()
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("response_bytes", max(c.Writer.Size(), 0)),
		}
		// The principal comes from the token, set by the authenticate middleware.
		if p, ok := auth.FromContext(ctx); ok {
			attrs = append(attrs, slog.String("user_id", p.UserId), slog.String("role", p.Role.String()))
			spanAttrs := []attribute.KeyValue{attribute.String("enduser.id", p.UserId)}
			if tenantId := p.EffectiveTenantId(); tenantId != "" {
				attrs = append(attrs, slog.String("tenant_id", tenantId))
				spanAttrs = append(spanAttrs, attribute.String("tenant.id", tenantId))
			}
			if p.IsImpersonating() {
				attrs = append(attrs, slog.Bool("impersonating", true))
			}
			trace.SpanFromContext(ctx).SetAttributes(spanAttrs...)
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case isHighVolume(route):
			level = slog.LevelDebug
		}
		slog.LogAttrs(ctx, level, "http request", attrs...)
	}
}

// isHighVolume tells the routes logged only at debug level: the pixel (one
// request per e-mail opened) and the health checks (every few seconds).
// Their volume is followed through the metrics.
func isHighVolume(route string) bool {
	return route == "/pixel/:campaignId" || strings.HasPrefix(route, "/health/")
}

// recoverPanic turns a panic into a 500 with the request id, logging the stack.
func recoverPanic() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if r == http.ErrAbortHandler {
				panic(r) // deliberate abort of the response, handled by net/http
			}
			ctx := c.Request.Context()
			slog.ErrorContext(ctx, "panic while serving request",
				"panic", r, "route", c.FullPath(), "stack", string(debug.Stack()))
			span := trace.SpanFromContext(ctx)
			span.SetStatus(codes.Error, "panic")
			span.AddEvent("panic", trace.WithAttributes(attribute.String("panic.value", slog.AnyValue(r).String())))

			if c.Writer.Written() {
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, internalError(ctx))
		}()
		c.Next()
	}
}

func internalError(ctx context.Context) errorResponse {
	return errorResponse{Error: "internal server error", RequestId: telemetry.RequestId(ctx)}
}

func (h *handler) live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ready answers 503 while the database does not answer, so a load balancer or
// orchestrator stops sending traffic. The error is logged, never returned.
func (h *handler) ready(c *gin.Context) {
	if h.readiness != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readyTimeout)
		defer cancel()
		if err := h.readiness(ctx); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
