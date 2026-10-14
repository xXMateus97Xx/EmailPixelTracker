package telemetry

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "email_tracker"

// Metrics are the Prometheus metrics of the application. All methods accept a
// nil receiver and do nothing, so metrics are optional (tests, tools).
//
// Labels never carry ids (campaign, user, externalUserId): each distinct value
// creates a new time series.
type Metrics struct {
	registry *prometheus.Registry

	httpRequests   *prometheus.CounterVec
	httpDuration   *prometheus.HistogramVec
	pixelRequests  *prometheus.CounterVec
	pixelCache     prometheus.Gauge
	logins         *prometheus.CounterVec
	impersonations *prometheus.CounterVec
	dbDuration     *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "http_requests_total",
			Help: "HTTP requests by route pattern, method and status code.",
		}, []string{"route", "method", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "http_request_duration_seconds",
			Help:    "HTTP request duration by route pattern and method.",
			Buckets: prometheus.DefBuckets,
		}, []string{"route", "method"}),
		pixelRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "pixel_requests_total",
			Help: "Pixel requests by result: tracked, or why the view was not tracked.",
		}, []string{"result"}),
		pixelCache: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Name: "pixel_cache_entries",
			Help: "Encoded pixel images kept in memory (one per distinct color).",
		}),
		logins: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "logins_total",
			Help: "Login attempts by result.",
		}, []string{"result"}),
		impersonations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "impersonations_total",
			Help: "Impersonations started and stopped by SuperAdmins.",
		}, []string{"action"}),
		dbDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "db_operation_duration_seconds",
			Help:    "Database operation duration by system, operation and status (ok or error).",
			Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"system", "operation", "status"}),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.httpRequests, m.httpDuration, m.pixelRequests, m.pixelCache,
		m.logins, m.impersonations, m.dbDuration,
	)
	return m
}

// Handler serves the metrics in the Prometheus exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}

// RegisterSQLStats exposes the connection pool statistics (open, in use,
// idle, waits) of a SQL database.
func (m *Metrics) RegisterSQLStats(db *sql.DB, name string) error {
	if m == nil {
		return nil
	}
	return m.registry.Register(collectors.NewDBStatsCollector(db, name))
}

// ObserveHTTP records a finished request. route is the route pattern
// (e.g. /pixel/:campaignId), never the raw path.
func (m *Metrics) ObserveHTTP(route, method string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	m.httpRequests.WithLabelValues(route, method, strconv.Itoa(status)).Inc()
	m.httpDuration.WithLabelValues(route, method).Observe(duration.Seconds())
}

func (m *Metrics) PixelRequest(result string) {
	if m != nil {
		m.pixelRequests.WithLabelValues(result).Inc()
	}
}

func (m *Metrics) PixelCached() {
	if m != nil {
		m.pixelCache.Inc()
	}
}

func (m *Metrics) Login(result string) {
	if m != nil {
		m.logins.WithLabelValues(result).Inc()
	}
}

// Impersonation records an impersonation action: start or stop.
func (m *Metrics) Impersonation(action string) {
	if m != nil {
		m.impersonations.WithLabelValues(action).Inc()
	}
}

func (m *Metrics) ObserveDB(system, operation string, duration time.Duration, failed bool) {
	if m == nil {
		return
	}
	status := "ok"
	if failed {
		status = "error"
	}
	m.dbDuration.WithLabelValues(system, operation, status).Observe(duration.Seconds())
}
