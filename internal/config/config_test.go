package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDefaultsInDev(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AppEnv != "dev" {
		t.Fatalf("AppEnv = %q, want dev", cfg.AppEnv)
	}
	if cfg.ListenAddr() != "0.0.0.0:8080" {
		t.Fatalf("ListenAddr = %q, want 0.0.0.0:8080", cfg.ListenAddr())
	}
	if cfg.JudgeCallbackToken != "dev-judge-callback-token" {
		t.Fatalf("JudgeCallbackToken = %q, want dev fallback", cfg.JudgeCallbackToken)
	}
	if cfg.JudgeAPIToken != "dev-judge-api-token" {
		t.Fatalf("JudgeAPIToken = %q, want dev fallback", cfg.JudgeAPIToken)
	}
	if cfg.SessionSecure {
		t.Fatal("SessionSecure = true, want false in dev by default")
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("AllowedOrigins = %#v", cfg.AllowedOrigins)
	}
}

func TestLoadNormalizesConfiguredValues(t *testing.T) {
	t.Setenv("DATABASE_URL", " postgres://db ")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("ALLOWED_ORIGINS", " https://app.example.com, ,https://admin.example.com ")
	t.Setenv("FRONTEND_PUBLIC_URL", " https://app.example.com/ ")
	t.Setenv("BACKEND_PUBLIC_URL", " https://api.example.com/ ")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("SESSION_COOKIE_SAME_SITE", "none")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("SUBMISSION_RATE_LIMIT", "25")
	t.Setenv("SUBMISSION_RATE_WINDOW", "30s")
	t.Setenv("MATCH_DURATION", "10m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ListenAddr() != "0.0.0.0:9090" {
		t.Fatalf("ListenAddr = %q", cfg.ListenAddr())
	}
	if got, want := cfg.FrontendBaseURL, "https://app.example.com"; got != want {
		t.Fatalf("FrontendBaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.BackendBaseURL, "https://api.example.com"; got != want {
		t.Fatalf("BackendBaseURL = %q, want %q", got, want)
	}
	if got := strings.Join(cfg.AllowedOrigins, ","); got != "https://app.example.com,https://admin.example.com" {
		t.Fatalf("AllowedOrigins = %q", got)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Fatalf("SessionTTL = %s, want 24h", cfg.SessionTTL)
	}
	if cfg.SubmissionRateLimit != 25 || cfg.SubmissionRateWindow != 30*time.Second {
		t.Fatalf("rate limit = %d/%s", cfg.SubmissionRateLimit, cfg.SubmissionRateWindow)
	}
	if cfg.MatchDuration != 10*time.Minute {
		t.Fatalf("MatchDuration = %s", cfg.MatchDuration)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := Load()
	if err == nil {
		t.Fatal("Load returned nil error, want DATABASE_URL error")
	}
}

func TestLoadValidatesProdSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("DATABASE_URL", "postgres://db")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JUDGE_CALLBACK_TOKEN") {
		t.Fatalf("Load error = %v, want missing JUDGE_CALLBACK_TOKEN", err)
	}

	t.Run("missing api token", func(t *testing.T) {
		t.Setenv("APP_ENV", "prod")
		t.Setenv("DATABASE_URL", "postgres://db")
		t.Setenv("JUDGE_CALLBACK_TOKEN", "callback")
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "JUDGE_API_TOKEN") {
			t.Fatalf("Load error = %v, want missing JUDGE_API_TOKEN", err)
		}
	})

	t.Run("missing resend key", func(t *testing.T) {
		t.Setenv("APP_ENV", "prod")
		t.Setenv("DATABASE_URL", "postgres://db")
		t.Setenv("JUDGE_CALLBACK_TOKEN", "callback")
		t.Setenv("JUDGE_API_TOKEN", "api")
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "RESEND_API_KEY") {
			t.Fatalf("Load error = %v, want missing RESEND_API_KEY", err)
		}
	})

	t.Run("missing email from", func(t *testing.T) {
		t.Setenv("APP_ENV", "prod")
		t.Setenv("DATABASE_URL", "postgres://db")
		t.Setenv("JUDGE_CALLBACK_TOKEN", "callback")
		t.Setenv("JUDGE_API_TOKEN", "api")
		t.Setenv("RESEND_API_KEY", "resend")
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "EMAIL_FROM") {
			t.Fatalf("Load error = %v, want missing EMAIL_FROM", err)
		}
	})
}

func TestLoadRequiresCompleteR2Config(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("R2_ACCOUNT_ID", "account")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "R2_ACCOUNT_ID") {
		t.Fatalf("Load error = %v, want incomplete R2 error", err)
	}
}

func TestLoadValidatesNumericAndDurationValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{name: "invalid session ttl", key: "SESSION_TTL", val: "bad", want: "SESSION_TTL"},
		{name: "invalid session cleanup", key: "SESSION_CLEANUP_INTERVAL", val: "bad", want: "SESSION_CLEANUP_INTERVAL"},
		{name: "invalid oauth state ttl", key: "OAUTH_STATE_TTL", val: "bad", want: "OAUTH_STATE_TTL"},
		{name: "invalid recaptcha timeout parse", key: "GOOGLE_RECAPTCHA_TIMEOUT", val: "bad", want: "GOOGLE_RECAPTCHA_TIMEOUT"},
		{name: "invalid recaptcha timeout", key: "GOOGLE_RECAPTCHA_TIMEOUT", val: "0s", want: "GOOGLE_RECAPTCHA_TIMEOUT must be > 0"},
		{name: "invalid recaptcha score", key: "GOOGLE_RECAPTCHA_MIN_SCORE", val: "2", want: "GOOGLE_RECAPTCHA_MIN_SCORE"},
		{name: "invalid secure cookie bool", key: "SESSION_COOKIE_SECURE", val: "maybe", want: "SESSION_COOKIE_SECURE"},
		{name: "invalid same site", key: "SESSION_COOKIE_SAME_SITE", val: "wild", want: "SESSION_COOKIE_SAME_SITE"},
		{name: "none same site requires secure", key: "SESSION_COOKIE_SAME_SITE", val: "none", want: "SESSION_COOKIE_SECURE"},
		{name: "invalid judge timeout", key: "JUDGE_TIMEOUT", val: "bad", want: "JUDGE_TIMEOUT"},
		{name: "invalid max source", key: "MAX_SOURCE_CODE_BYTES", val: "0", want: "MAX_SOURCE_CODE_BYTES"},
		{name: "invalid max source parse", key: "MAX_SOURCE_CODE_BYTES", val: "bad", want: "MAX_SOURCE_CODE_BYTES"},
		{name: "invalid max body", key: "HTTP_MAX_BODY_BYTES", val: "0", want: "HTTP_MAX_BODY_BYTES"},
		{name: "invalid max body parse", key: "HTTP_MAX_BODY_BYTES", val: "bad", want: "HTTP_MAX_BODY_BYTES"},
		{name: "invalid submission limit", key: "SUBMISSION_RATE_LIMIT", val: "0", want: "SUBMISSION_RATE_LIMIT"},
		{name: "invalid submission limit parse", key: "SUBMISSION_RATE_LIMIT", val: "bad", want: "SUBMISSION_RATE_LIMIT"},
		{name: "invalid submission window parse", key: "SUBMISSION_RATE_WINDOW", val: "bad", want: "SUBMISSION_RATE_WINDOW"},
		{name: "invalid submission window", key: "SUBMISSION_RATE_WINDOW", val: "0s", want: "SUBMISSION_RATE_WINDOW"},
		{name: "invalid match duration parse", key: "MATCH_DURATION", val: "bad", want: "MATCH_DURATION"},
		{name: "invalid match duration", key: "MATCH_DURATION", val: "0s", want: "MATCH_DURATION"},
		{name: "invalid match cleanup parse", key: "MATCH_CLEANUP_INTERVAL", val: "bad", want: "MATCH_CLEANUP_INTERVAL"},
		{name: "invalid match cleanup", key: "MATCH_CLEANUP_INTERVAL", val: "0s", want: "MATCH_CLEANUP_INTERVAL"},
		{name: "invalid pending timeout parse", key: "MATCH_PENDING_TIMEOUT", val: "bad", want: "MATCH_PENDING_TIMEOUT"},
		{name: "invalid pending timeout", key: "MATCH_PENDING_TIMEOUT", val: "0s", want: "MATCH_PENDING_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://db")
			t.Setenv(tt.key, tt.val)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestConfigR2Enabled(t *testing.T) {
	disabled := Config{R2AccountID: "account"}
	if disabled.R2Enabled() {
		t.Fatal("R2Enabled = true for incomplete config")
	}

	enabled := Config{
		R2AccountID:       "account",
		R2AccessKeyID:     "key",
		R2SecretAccessKey: "secret",
		R2Bucket:          "bucket",
		R2PublicURL:       "https://cdn.example.com",
	}
	if !enabled.R2Enabled() {
		t.Fatal("R2Enabled = false for complete config")
	}
}

func TestParseHelpers(t *testing.T) {
	t.Setenv("BOOL_VALUE", "true")
	gotBool, err := parseBoolWithDefault("BOOL_VALUE", false)
	if err != nil || !gotBool {
		t.Fatalf("parseBoolWithDefault = %v, %v", gotBool, err)
	}

	t.Setenv("INT_VALUE", "42")
	gotInt, err := parseIntWithDefault("INT_VALUE", 0)
	if err != nil || gotInt != 42 {
		t.Fatalf("parseIntWithDefault = %d, %v", gotInt, err)
	}

	t.Setenv("FLOAT_VALUE", "0.75")
	gotFloat, err := parseFloatWithDefault("FLOAT_VALUE", 0)
	if err != nil || gotFloat != 0.75 {
		t.Fatalf("parseFloatWithDefault = %f, %v", gotFloat, err)
	}

	if _, err := parseBoolWithDefault("MISSING_BOOL", true); err != nil {
		t.Fatalf("missing bool default returned error: %v", err)
	}

	t.Setenv("BAD_BOOL", "maybe")
	if _, err := parseBoolWithDefault("BAD_BOOL", false); err == nil {
		t.Fatal("parseBoolWithDefault returned nil error for invalid bool")
	}

	t.Setenv("BAD_INT", "nope")
	if _, err := parseIntWithDefault("BAD_INT", 0); err == nil {
		t.Fatal("parseIntWithDefault returned nil error for invalid int")
	}

	t.Setenv("BAD_FLOAT", "nope")
	if _, err := parseFloatWithDefault("BAD_FLOAT", 0); err == nil {
		t.Fatal("parseFloatWithDefault returned nil error for invalid float")
	}

	if _, err := parseSameSiteWithDefault("MISSING_SAME_SITE", "lax"); err != nil {
		t.Fatalf("parseSameSiteWithDefault returned error: %v", err)
	}
}
