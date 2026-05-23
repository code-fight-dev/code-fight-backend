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
	}

	m.registry.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpInFlight,
		m.avatarUploadsTotal,
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
