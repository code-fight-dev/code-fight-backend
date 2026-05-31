package metrics

import (
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestBackendMetricsRecordsBusinessCounters(t *testing.T) {
	m := NewBackendMetrics()

	m.RecordAvatarUpload("success")
	m.RecordSubmissionCreated("task", "go", "success")
	m.RecordSubmissionFinished("match", "go", "finished", "accepted")
	m.RecordCodeRunCreated("go", "success")
	m.RecordCodeRunFinished("go", "failed", "none")
	m.RecordMatchCreated("normal", "global", true)
	m.RecordMatchFinished("draw", "draw", false)
	m.RecordMatchCancelled("pending_timeout")

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	names := make(map[string]bool)
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, name := range []string{
		"codefight_backend_avatar_uploads_total",
		"codefight_backend_submissions_created_total",
		"codefight_backend_submissions_finished_total",
		"codefight_backend_code_runs_created_total",
		"codefight_backend_code_runs_finished_total",
		"codefight_backend_matches_created_total",
		"codefight_backend_matches_finished_total",
		"codefight_backend_matches_cancelled_total",
	} {
		if !names[name] {
			t.Fatalf("missing metric %s", name)
		}
	}
}

func TestBackendMetricsMiddlewareAndHandler(t *testing.T) {
	m := NewBackendMetrics()
	app := fiber.New()
	app.Use(m.Middleware())
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/fail", func(c *fiber.Ctx) error { return fiber.NewError(fiber.StatusTeapot, "fail") })
	app.Get("/metrics", m.Handler())

	for _, path := range []string{"/ok", "/fail", "/metrics"} {
		req, err := nethttp.NewRequest(nethttp.MethodGet, path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app test %s: %v", path, err)
		}
		_ = resp.Body.Close()
	}

	req, _ := nethttp.NewRequest(nethttp.MethodGet, "/metrics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("metrics request: %v", err)
	}
	defer resp.Body.Close()

	buf := new(strings.Builder)
	_, _ = io.Copy(buf, resp.Body)
	if !strings.Contains(buf.String(), "codefight_backend_http_requests_total") {
		t.Fatalf("metrics output missing http counter: %s", buf.String())
	}
}
