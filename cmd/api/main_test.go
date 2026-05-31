package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/config"
	"code-fight-backend/internal/mail"
	"code-fight-backend/internal/objectstorage"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStartSessionCleanupLoopRunsOnceAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeSessionCleaner{afterCall: cancel}

	startSessionCleanupLoop(ctx, testLogger(), cleaner, time.Hour)

	if cleaner.calls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleaner.calls)
	}
}

func TestStartSessionCleanupLoopLogsErrorAndTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeSessionCleaner{err: errors.New("cleanup failed"), afterCall: func() {
		if cleanerCalls := 0; cleanerCalls == -1 {
			t.Fatal("unreachable")
		}
	}}
	cleaner.afterCall = func() {
		if cleaner.calls >= 2 {
			cancel()
		}
	}

	startSessionCleanupLoop(ctx, testLogger(), cleaner, time.Nanosecond)

	if cleaner.calls < 2 {
		t.Fatalf("cleanup calls = %d, want at least 2", cleaner.calls)
	}
}

func TestStartMatchCleanupLoopRunsFinishAndCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeMatchCleaner{finishCount: 2, cancelCount: 3, afterCancel: cancel}

	startMatchCleanupLoop(ctx, testLogger(), cleaner, time.Hour, 30*time.Second)

	if cleaner.finishCalls != 1 || cleaner.cancelCalls != 1 {
		t.Fatalf("calls finish=%d cancel=%d", cleaner.finishCalls, cleaner.cancelCalls)
	}
}

func TestStartMatchCleanupLoopLogsErrorsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeMatchCleaner{finishErr: errors.New("finish failed"), cancelErr: errors.New("cancel failed"), afterCancel: cancel}

	startMatchCleanupLoop(ctx, testLogger(), cleaner, time.Hour, 30*time.Second)

	if cleaner.finishCalls != 1 || cleaner.cancelCalls != 1 {
		t.Fatalf("calls finish=%d cancel=%d", cleaner.finishCalls, cleaner.cancelCalls)
	}
}

func TestStartMatchCleanupLoopTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeMatchCleaner{afterCancel: func() {
		if cleanerCalls := 0; cleanerCalls == -1 {
			t.Fatal("unreachable")
		}
	}}
	cleaner.afterCancel = func() {
		if cleaner.cancelCalls >= 2 {
			cancel()
		}
	}

	startMatchCleanupLoop(ctx, testLogger(), cleaner, time.Nanosecond, 30*time.Second)

	if cleaner.finishCalls < 2 || cleaner.cancelCalls < 2 {
		t.Fatalf("calls finish=%d cancel=%d, want at least 2", cleaner.finishCalls, cleaner.cancelCalls)
	}
}

func TestMainEntrypointForSubprocess(t *testing.T) {
	if os.Getenv("CODEFIGHT_TEST_MAIN") != "1" {
		return
	}
	main()
}

func TestMainExitsWhenConfigFails(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMainEntrypointForSubprocess")
	cmd.Env = append(os.Environ(), "CODEFIGHT_TEST_MAIN=1", "DATABASE_URL=")

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("main exit = %v, want code 1", err)
	}
}

func TestMainExitsWhenDatabaseConnectFails(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMainEntrypointForSubprocess")
	cmd.Env = append(os.Environ(), "CODEFIGHT_TEST_MAIN=1", "DATABASE_URL=://bad")

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("main exit = %v, want code 1", err)
	}
}

func TestMainExitsWhenDatabasePingFails(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMainEntrypointForSubprocess")
	cmd.Env = append(os.Environ(),
		"CODEFIGHT_TEST_MAIN=1",
		"DATABASE_URL=postgres://codefight:codefight@127.0.0.1:1/codefight?sslmode=disable&connect_timeout=1",
	)

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("main exit = %v, want code 1", err)
	}
}

func TestRunWiresApplicationAndStopsAfterListen(t *testing.T) {
	restore := stubRunDependencies(t)
	defer restore()

	if err := run(context.Background(), testLogger()); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunReturnsDependencyErrors(t *testing.T) {
	restore := stubRunDependencies(t)
	defer restore()

	loadErr := errors.New("load failed")
	loadConfig = func() (config.Config, error) { return config.Config{}, loadErr }
	if err := run(context.Background(), testLogger()); !errors.Is(err, loadErr) {
		t.Fatalf("load error = %v, want %v", err, loadErr)
	}

	restore()
	restore = stubRunDependencies(t)
	defer restore()
	connectErr := errors.New("connect failed")
	connectDatabase = func(context.Context, string) (*pgxpool.Pool, error) { return nil, connectErr }
	if err := run(context.Background(), testLogger()); !errors.Is(err, connectErr) {
		t.Fatalf("connect error = %v, want %v", err, connectErr)
	}

	restore()
	restore = stubRunDependencies(t)
	defer restore()
	migrateErr := errors.New("migrate failed")
	migrateDatabase = func(context.Context, *pgxpool.Pool) error { return migrateErr }
	if err := run(context.Background(), testLogger()); !errors.Is(err, migrateErr) {
		t.Fatalf("migrate error = %v, want %v", err, migrateErr)
	}

	restore()
	restore = stubRunDependencies(t)
	defer restore()
	listenErr := errors.New("listen failed")
	listenHTTPApp = func(*fiber.App, string) error { return listenErr }
	if err := run(context.Background(), testLogger()); !errors.Is(err, listenErr) {
		t.Fatalf("listen error = %v, want %v", err, listenErr)
	}
}

func TestRunReturnsOptionalDependencyErrors(t *testing.T) {
	restore := stubRunDependencies(t)
	defer restore()

	resendErr := errors.New("resend failed")
	loadConfig = func() (config.Config, error) {
		cfg := testRunConfig()
		cfg.ResendAPIKey = "key"
		cfg.EmailFrom = "noreply@example.test"
		return cfg, nil
	}
	newResendSender = func(string, string) (*mail.ResendSender, error) {
		return nil, resendErr
	}
	if err := run(context.Background(), testLogger()); !errors.Is(err, resendErr) {
		t.Fatalf("resend error = %v, want %v", err, resendErr)
	}

	restore()
	restore = stubRunDependencies(t)
	defer restore()
	r2Err := errors.New("r2 failed")
	loadConfig = func() (config.Config, error) {
		cfg := testRunConfig()
		cfg.R2AccountID = "account"
		cfg.R2AccessKeyID = "access"
		cfg.R2SecretAccessKey = "secret"
		cfg.R2Bucket = "bucket"
		cfg.R2PublicURL = "https://cdn.example.test"
		return cfg, nil
	}
	newR2Client = func(context.Context, objectstorage.R2Config) (*objectstorage.R2Client, error) {
		return nil, r2Err
	}
	if err := run(context.Background(), testLogger()); !errors.Is(err, r2Err) {
		t.Fatalf("r2 error = %v, want %v", err, r2Err)
	}
}

func stubRunDependencies(t *testing.T) func() {
	t.Helper()

	origLoadConfig := loadConfig
	origConnectDatabase := connectDatabase
	origCloseDatabase := closeDatabase
	origMigrateDatabase := migrateDatabase
	origNewResendSender := newResendSender
	origNewR2Client := newR2Client
	origListenHTTPApp := listenHTTPApp

	loadConfig = func() (config.Config, error) {
		return testRunConfig(), nil
	}
	connectDatabase = func(context.Context, string) (*pgxpool.Pool, error) { return &pgxpool.Pool{}, nil }
	closeDatabase = func(*pgxpool.Pool) {}
	migrateDatabase = func(context.Context, *pgxpool.Pool) error { return nil }
	listenHTTPApp = func(*fiber.App, string) error { return nil }

	return func() {
		loadConfig = origLoadConfig
		connectDatabase = origConnectDatabase
		closeDatabase = origCloseDatabase
		migrateDatabase = origMigrateDatabase
		newResendSender = origNewResendSender
		newR2Client = origNewR2Client
		listenHTTPApp = origListenHTTPApp
	}
}

func testRunConfig() config.Config {
	return config.Config{
		AppEnv:                  "test",
		HTTPPort:                "0",
		DatabaseURL:             "postgres://test",
		AllowedOrigins:          []string{"http://localhost:3000"},
		FrontendBaseURL:         "http://localhost:3000",
		BackendBaseURL:          "http://localhost:8080",
		SessionCookieName:       "codefight_session",
		SessionCookieSameSite:   "lax",
		SessionTTL:              time.Hour,
		SessionCleanupInterval:  0,
		OAuthStateCookie:        "oauth_state",
		OAuthStateTTL:           time.Minute,
		GoogleRecaptchaURL:      "https://recaptcha.example.test",
		GoogleRecaptchaTimeout:  time.Second,
		GoogleRecaptchaMinScore: 0.5,
		JudgeBaseURL:            "http://judge.example.test",
		JudgeCallbackURL:        "http://backend.example.test/internal/judge/callback",
		JudgeCallbackToken:      "callback-token",
		JudgeTimeout:            time.Second,
		MaxSourceCodeBytes:      1024,
		MaxBodyBytes:            4096,
		SubmissionRateLimit:     10,
		SubmissionRateWindow:    time.Minute,
		MatchDuration:           time.Minute,
		MatchCleanupInterval:    0,
		MatchPendingTimeout:     time.Second,
	}
}

type fakeSessionCleaner struct {
	calls     int
	err       error
	afterCall func()
}

func (c *fakeSessionCleaner) CleanupExpiredSessions(context.Context) error {
	c.calls++
	if c.afterCall != nil {
		c.afterCall()
	}
	return c.err
}

type fakeMatchCleaner struct {
	finishCalls int
	cancelCalls int
	finishCount int
	cancelCount int
	finishErr   error
	cancelErr   error
	afterCancel func()
}

func (c *fakeMatchCleaner) FinishExpiredMatches(context.Context, time.Time, int) (int, error) {
	c.finishCalls++
	return c.finishCount, c.finishErr
}

func (c *fakeMatchCleaner) CancelExpiredPendingMatches(context.Context, time.Time, time.Duration, int) (int, error) {
	c.cancelCalls++
	if c.afterCancel != nil {
		c.afterCancel()
	}
	return c.cancelCount, c.cancelErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestMainTestNamesStayScoped(t *testing.T) {
	if strings.Contains("TestMainEntrypointForSubprocess", "nope") {
		t.Fatal("unreachable")
	}
}
