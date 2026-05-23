package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type BackendMetrics struct {
	registry *prometheus.Registry

	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpInFlight        prometheus.Gauge
	avatarUploadsTotal  *prometheus.CounterVec
	submissionsCreated  *prometheus.CounterVec
	submissionsFinished *prometheus.CounterVec
	codeRunsCreated     *prometheus.CounterVec
	codeRunsFinished    *prometheus.CounterVec
	matchesCreated      *prometheus.CounterVec
	matchesFinished     *prometheus.CounterVec
	matchesCancelled    *prometheus.CounterVec
}

func NewBackendMetrics() *BackendMetrics {
	m := &BackendMetrics{
		registry: prometheus.NewRegistry(),
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_http_requests_total",
				Help: "Total number of backend HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "codefight_backend_http_request_duration_seconds",
				Help:    "Backend HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		httpInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "codefight_backend_http_in_flight_requests",
				Help: "Current number of in-flight backend HTTP requests.",
			},
		),
		avatarUploadsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_avatar_uploads_total",
				Help: "Total number of backend avatar upload attempts.",
			},
			[]string{"status"},
		),
		submissionsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_submissions_created_total",
				Help: "Total number of task and match submissions created.",
			},
			[]string{"scope", "language", "status"},
		),
		submissionsFinished: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_submissions_finished_total",
				Help: "Total number of task and match submissions completed by judge callback.",
			},
			[]string{"scope", "language", "status", "verdict"},
		),
		codeRunsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_code_runs_created_total",
				Help: "Total number of code runs created.",
			},
			[]string{"language", "status"},
		),
		codeRunsFinished: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_code_runs_finished_total",
				Help: "Total number of code runs completed by judge callback.",
			},
			[]string{"language", "status", "verdict"},
		),
		matchesCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_matches_created_total",
				Help: "Total number of matches created.",
			},
			[]string{"task_mode", "rating_mode", "rated"},
		),
		matchesFinished: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_matches_finished_total",
				Help: "Total number of matches finished.",
			},
			[]string{"result_type", "winning_reason", "rated"},
		),
		matchesCancelled: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "codefight_backend_matches_cancelled_total",
				Help: "Total number of matches cancelled.",
			},
			[]string{"source"},
		),
	}

	m.registry.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpInFlight,
		m.avatarUploadsTotal,
		m.submissionsCreated,
		m.submissionsFinished,
		m.codeRunsCreated,
		m.codeRunsFinished,
		m.matchesCreated,
		m.matchesFinished,
		m.matchesCancelled,
	)

	return m
}

func (m *BackendMetrics) Handler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
}

func (m *BackendMetrics) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/metrics" {
			return c.Next()
		}

		startedAt := time.Now()
		m.httpInFlight.Inc()
		err := c.Next()
		m.httpInFlight.Dec()

		path := c.Path()
		if route := c.Route(); route != nil && route.Path != "" {
			path = route.Path
		}

		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		method := c.Method()
		m.httpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
		m.httpRequestDuration.WithLabelValues(method, path).Observe(time.Since(startedAt).Seconds())

		return err
	}
}

func (m *BackendMetrics) RecordAvatarUpload(status string) {
	m.avatarUploadsTotal.WithLabelValues(status).Inc()
}

func (m *BackendMetrics) RecordSubmissionCreated(scope string, language string, status string) {
	m.submissionsCreated.WithLabelValues(scope, language, status).Inc()
}

func (m *BackendMetrics) RecordSubmissionFinished(scope string, language string, status string, verdict string) {
	m.submissionsFinished.WithLabelValues(scope, language, status, verdict).Inc()
}

func (m *BackendMetrics) RecordCodeRunCreated(language string, status string) {
	m.codeRunsCreated.WithLabelValues(language, status).Inc()
}

func (m *BackendMetrics) RecordCodeRunFinished(language string, status string, verdict string) {
	m.codeRunsFinished.WithLabelValues(language, status, verdict).Inc()
}

func (m *BackendMetrics) RecordMatchCreated(taskMode string, ratingMode string, isRated bool) {
	m.matchesCreated.WithLabelValues(taskMode, ratingMode, strconv.FormatBool(isRated)).Inc()
}

func (m *BackendMetrics) RecordMatchFinished(resultType string, winningReason string, isRated bool) {
	m.matchesFinished.WithLabelValues(resultType, winningReason, strconv.FormatBool(isRated)).Inc()
}

func (m *BackendMetrics) RecordMatchCancelled(source string) {
	m.matchesCancelled.WithLabelValues(source).Inc()
}
