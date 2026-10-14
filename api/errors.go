package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xXMateus97Xx/email-pixel-tracker/service"
)

type fieldErrorResponse struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error  string               `json:"error"`
	Fields []fieldErrorResponse `json:"fields,omitempty"`
	// RequestId is sent with internal errors, so they can be found in the logs.
	RequestId string `json:"requestId,omitempty"`
}

func writeError(c *gin.Context, err error) {
	var verr *service.ValidationError
	if errors.As(err, &verr) {
		fields := make([]fieldErrorResponse, len(verr.Errors))
		for i, fe := range verr.Errors {
			fields[i] = fieldErrorResponse{Field: fe.Field, Message: fe.Message}
		}
		c.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrValidation.Error(), Fields: fields})
		return
	}

	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrUnauthenticated), errors.Is(err, service.ErrInvalidCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrImpersonationRequired),
		errors.Is(err, service.ErrTenantInactive):
		status = http.StatusForbidden
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
	}

	if status == http.StatusInternalServerError {
		// The details stay in the logs and the trace; the client gets the request id.
		ctx := c.Request.Context()
		slog.ErrorContext(ctx, "request failed", "route", c.FullPath(), "error", err)
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(status, internalError(ctx))
		return
	}
	c.JSON(status, errorResponse{Error: err.Error()})
}

func badRequest(c *gin.Context, message string, fields ...fieldErrorResponse) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: message, Fields: fields})
}

// bindJSON decodes the body, answering 400 on malformed JSON.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		badRequest(c, "invalid request body: "+err.Error())
		return false
	}
	return true
}

// pageFromQuery reads ?page=&pageSize=; the services apply defaults and limits.
func pageFromQuery(c *gin.Context) (service.Page, bool) {
	var page service.Page
	for _, q := range []struct {
		name string
		dst  *int
	}{{"page", &page.Number}, {"pageSize", &page.Size}} {
		raw := c.Query(q.name)
		if raw == "" {
			continue
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			badRequest(c, "invalid query parameter", fieldErrorResponse{Field: q.name, Message: "must be a positive integer"})
			return page, false
		}
		*q.dst = n
	}
	return page, true
}
