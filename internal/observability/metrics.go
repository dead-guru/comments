package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultNamespace = "deadcomments"

type Metrics struct {
	registry *prometheus.Registry

	httpInFlight     *prometheus.GaugeVec
	httpRequests     *prometheus.CounterVec
	httpDuration     *prometheus.HistogramVec
	httpResponseSize *prometheus.HistogramVec
	readinessChecks  *prometheus.CounterVec

	domainEvents      *prometheus.CounterVec
	commentsCreated   *prometheus.CounterVec
	commentModeration *prometheus.CounterVec
	pageEvents        *prometheus.CounterVec
	siteEvents        *prometheus.CounterVec
	banEvents         *prometheus.CounterVec
	identityEvents    *prometheus.CounterVec
	adminLogins       prometheus.Counter
}

func NewMetrics(namespace string) *Metrics {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = defaultNamespace
	}

	m := &Metrics{
		registry: prometheus.NewRegistry(),
		httpInFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Current number of in-flight HTTP requests.",
		}, []string{"method"}),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		}, []string{"method", "route", "code"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route", "code"}),
		httpResponseSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes.",
			Buckets:   []float64{100, 500, 1_000, 5_000, 10_000, 50_000, 100_000, 500_000, 1_000_000},
		}, []string{"method", "route", "code"}),
		readinessChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "readiness_checks_total",
			Help:      "Readiness check results by component.",
		}, []string{"component", "result"}),
		domainEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "domain_events_total",
			Help:      "Durable domain events published by the application.",
		}, []string{"event_type", "aggregate_type"}),
		commentsCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "comments_created_total",
			Help:      "Comments created by moderation status, reason, tripcode kind, and reply state.",
		}, []string{"status", "reason", "tripcode_kind", "has_parent"}),
		commentModeration: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "comment_moderation_actions_total",
			Help:      "Admin moderation transitions for comments.",
		}, []string{"action", "from_status", "to_status"}),
		pageEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "page_events_total",
			Help:      "Page lifecycle events.",
		}, []string{"action", "state"}),
		siteEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "site_events_total",
			Help:      "Site lifecycle events.",
		}, []string{"action"}),
		banEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ban_events_total",
			Help:      "IP and word-ban lifecycle events.",
		}, []string{"kind", "action"}),
		identityEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "identity_events_total",
			Help:      "Reserved identity lifecycle events.",
		}, []string{"action", "badge_type"}),
		adminLogins: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "admin_logins_total",
			Help:      "Successful admin logins.",
		}),
	}

	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),
		m.httpInFlight,
		m.httpRequests,
		m.httpDuration,
		m.httpResponseSize,
		m.readinessChecks,
		m.domainEvents,
		m.commentsCreated,
		m.commentModeration,
		m.pageEvents,
		m.siteEvents,
		m.banEvents,
		m.identityEvents,
		m.adminLogins,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          m.registry,
	})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		start := time.Now()
		m.httpInFlight.WithLabelValues(method).Inc()
		defer m.httpInFlight.WithLabelValues(method).Dec()

		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		code := ww.Status()
		if code == 0 {
			code = http.StatusOK
		}
		route := routePattern(r, code)
		codeLabel := strconv.Itoa(code)
		labels := prometheus.Labels{"method": method, "route": route, "code": codeLabel}

		m.httpRequests.With(labels).Inc()
		m.httpDuration.With(labels).Observe(time.Since(start).Seconds())
		m.httpResponseSize.With(labels).Observe(float64(ww.BytesWritten()))
	})
}

func (m *Metrics) RecordReadiness(component string, ok bool) {
	result := "ok"
	if !ok {
		result = "fail"
	}
	m.readinessChecks.WithLabelValues(cleanLabel(component, "unknown"), result).Inc()
}

func routePattern(r *http.Request, code int) string {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if code == http.StatusNotFound {
		return "unmatched"
	}
	return "unknown"
}

func cleanLabel(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	return value
}
