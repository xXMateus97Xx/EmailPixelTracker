package telemetry

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// GormPlugin traces and measures every GORM operation. system is the
// database driver (postgres, mysql, sqlserver). Spans carry the SQL with
// placeholders, never the values, which include password hashes and
// personal data.
func GormPlugin(m *Metrics, system string) gorm.Plugin {
	return &gormPlugin{metrics: m, system: system, tracer: otel.Tracer(instrumentationName)}
}

type gormPlugin struct {
	metrics *Metrics
	system  string
	tracer  trace.Tracer
}

func (p *gormPlugin) Name() string { return "telemetry" }

const gormStateKey = "telemetry:state"

// gormState is what the before callback hands to the after callback.
type gormState struct {
	start  time.Time
	span   trace.Span
	parent context.Context
	// own is false when no span was started (no parent span).
	own bool
}

// hasParentSpan limits the database spans to work done inside a trace, such
// as a request; startup migrations and health check pings would only be noise.
func hasParentSpan(ctx context.Context) bool {
	return trace.SpanContextFromContext(ctx).IsValid()
}

func (p *gormPlugin) Initialize(db *gorm.DB) error {
	cb := db.Callback()
	processors := []struct {
		operation string
		before    func(name string, fn func(*gorm.DB)) error
		after     func(name string, fn func(*gorm.DB)) error
	}{
		{"create", cb.Create().Before("gorm:create").Register, cb.Create().After("gorm:create").Register},
		{"query", cb.Query().Before("gorm:query").Register, cb.Query().After("gorm:query").Register},
		{"update", cb.Update().Before("gorm:update").Register, cb.Update().After("gorm:update").Register},
		{"delete", cb.Delete().Before("gorm:delete").Register, cb.Delete().After("gorm:delete").Register},
		{"row", cb.Row().Before("gorm:row").Register, cb.Row().After("gorm:row").Register},
		{"raw", cb.Raw().Before("gorm:raw").Register, cb.Raw().After("gorm:raw").Register},
	}
	for _, proc := range processors {
		if err := proc.before("telemetry:before_"+proc.operation, p.start(proc.operation)); err != nil {
			return err
		}
		if err := proc.after("telemetry:after_"+proc.operation, p.finish(proc.operation)); err != nil {
			return err
		}
	}
	return nil
}

func (p *gormPlugin) start(operation string) func(*gorm.DB) {
	return func(tx *gorm.DB) {
		parent := tx.Statement.Context
		ctx, span := parent, trace.SpanFromContext(parent)
		own := hasParentSpan(parent)
		if own {
			ctx, span = p.tracer.Start(parent, "gorm."+operation,
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(
					attribute.String("db.system.name", p.system),
					attribute.String("db.operation.name", operation),
				))
		}
		// The span becomes the parent of the slow query and error logs.
		tx.Statement.Context = ctx
		tx.InstanceSet(gormStateKey, &gormState{start: time.Now(), span: span, parent: parent, own: own})
	}
}

func (p *gormPlugin) finish(operation string) func(*gorm.DB) {
	return func(tx *gorm.DB) {
		v, ok := tx.InstanceGet(gormStateKey)
		if !ok {
			return
		}
		state := v.(*gormState)
		tx.Statement.Context = state.parent

		// Not finding a record is an expected outcome, not a database error.
		failed := tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound)
		p.metrics.ObserveDB(p.system, operation, time.Since(state.start), failed)

		if !state.own {
			return
		}
		span := state.span
		if tx.Statement.Table != "" {
			span.SetAttributes(attribute.String("db.collection.name", tx.Statement.Table))
		}
		span.SetAttributes(
			attribute.String("db.query.text", tx.Statement.SQL.String()),
			attribute.Int64("db.response.returned_rows", tx.Statement.RowsAffected),
		)
		if failed {
			span.RecordError(tx.Error)
			span.SetStatus(codes.Error, tx.Error.Error())
		}
		span.End()
	}
}

// MongoMonitor traces and measures every MongoDB command. Spans carry the
// database, collection and command name, never the command body, which may
// contain personal data.
func MongoMonitor(m *Metrics) *event.CommandMonitor {
	tracer := otel.Tracer(instrumentationName)
	var inFlight sync.Map // commandKey -> trace.Span

	finish := func(e event.CommandFinishedEvent, failure error) {
		m.ObserveDB("mongodb", e.CommandName, e.Duration, failure != nil)
		if v, ok := inFlight.LoadAndDelete(commandKey(e.ConnectionID, e.RequestID)); ok {
			span := v.(trace.Span)
			if failure != nil {
				span.RecordError(failure)
				span.SetStatus(codes.Error, failure.Error())
			}
			span.End()
		}
	}

	return &event.CommandMonitor{
		Started: func(ctx context.Context, e *event.CommandStartedEvent) {
			if !hasParentSpan(ctx) {
				return
			}
			collection, _ := e.Command.Lookup(e.CommandName).StringValueOK()
			name := e.CommandName
			if collection != "" {
				name += " " + collection
			}
			_, span := tracer.Start(ctx, name,
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(
					attribute.String("db.system.name", "mongodb"),
					attribute.String("db.namespace", e.DatabaseName),
					attribute.String("db.collection.name", collection),
					attribute.String("db.operation.name", e.CommandName),
				))
			inFlight.Store(commandKey(e.ConnectionID, e.RequestID), span)
		},
		Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
			finish(e.CommandFinishedEvent, nil)
		},
		Failed: func(_ context.Context, e *event.CommandFailedEvent) {
			finish(e.CommandFinishedEvent, e.Failure)
		},
	}
}

func commandKey(connectionId string, requestId int64) string {
	return connectionId + "/" + strconv.FormatInt(requestId, 10)
}
