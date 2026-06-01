package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPPort               = "8080"
	defaultSessionCookieName      = "codefight_session"
	defaultSessionCookieSameSite  = "lax"
	defaultSessionTTL             = "720h"
	defaultSessionCleanupInterval = "1h"
	defaultAllowedOrigins         = "http://localhost:3000"
	defaultFrontendBaseURL        = "http://localhost:3000"
	defaultBackendBaseURL         = "http://localhost:8080"
	defaultOAuthStateCookie       = "codefight_oauth_state"
	defaultOAuthStateTTL          = "10m"
	defaultGoogleRecaptchaURL     = "https://www.google.com/recaptcha/api/siteverify"
	defaultGoogleRecaptchaTimeout = "5s"

	defaultJudgeBaseURL         = "http://localhost:8090"
	defaultJudgeTimeout         = "10s"
	defaultMaxSourceCodeBytes   = 65536
	defaultMaxBodyBytes         = 4 << 20
	defaultSubmissionRateLimit  = 10
	defaultSubmissionRateWindow = "1m"
	defaultMatchDuration        = "20m"
	defaultMatchCleanupInterval = "30s"
	defaultMatchPendingTimeout  = "30s"
)

type Config struct {
	AppEnv                  string
	HTTPPort                string
	DatabaseURL             string
	AllowedOrigins          []string
	FrontendBaseURL         string
	BackendBaseURL          string
	SessionCookieName       string
	SessionCookieDomain     string
	SessionCookieSameSite   string
	SessionTTL              time.Duration
	SessionCleanupInterval  time.Duration
	SessionSecure           bool
	OAuthStateCookie        string
	OAuthStateTTL           time.Duration
	GitHubClientID          string
	GitHubSecret            string
	GoogleClientID          string
	GoogleSecret            string
	GoogleRecaptchaSecret   string
	GoogleRecaptchaURL      string
	GoogleRecaptchaTimeout  time.Duration
	GoogleRecaptchaMinScore float64
	GoogleRecaptchaAction   string
	ResendAPIKey            string
	EmailFrom               string
	R2AccountID             string
	R2AccessKeyID           string
	R2SecretAccessKey       string
	R2Bucket                string
	R2PublicURL             string

	JudgeBaseURL         string
	JudgeCallbackURL     string
	JudgeCallbackToken   string
	JudgeAPIToken        string
	JudgeTimeout         time.Duration
	MaxSourceCodeBytes   int
	MaxBodyBytes         int
	SubmissionRateLimit  int
	SubmissionRateWindow time.Duration
	MatchDuration        time.Duration
	MatchCleanupInterval time.Duration
	MatchPendingTimeout  time.Duration
}

func Load() (Config, error) {
	appEnv := valueOrDefault("APP_ENV", "dev")
	httpPort := valueOrDefault("HTTP_PORT", defaultHTTPPort)

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	sessionTTL, err := time.ParseDuration(valueOrDefault("SESSION_TTL", defaultSessionTTL))
	if err != nil {
		return Config{}, fmt.Errorf("parse SESSION_TTL: %w", err)
	}

	sessionCleanupInterval, err := time.ParseDuration(valueOrDefault("SESSION_CLEANUP_INTERVAL", defaultSessionCleanupInterval))
	if err != nil {
		return Config{}, fmt.Errorf("parse SESSION_CLEANUP_INTERVAL: %w", err)
	}

	oauthStateTTL, err := time.ParseDuration(valueOrDefault("OAUTH_STATE_TTL", defaultOAuthStateTTL))
	if err != nil {
		return Config{}, fmt.Errorf("parse OAUTH_STATE_TTL: %w", err)
	}

	googleRecaptchaTimeout, err := time.ParseDuration(valueOrDefault("GOOGLE_RECAPTCHA_TIMEOUT", defaultGoogleRecaptchaTimeout))
	if err != nil {
		return Config{}, fmt.Errorf("parse GOOGLE_RECAPTCHA_TIMEOUT: %w", err)
	}
	if googleRecaptchaTimeout <= 0 {
		return Config{}, fmt.Errorf("GOOGLE_RECAPTCHA_TIMEOUT must be > 0")
	}

	googleRecaptchaMinScore, err := parseFloatWithDefault("GOOGLE_RECAPTCHA_MIN_SCORE", 0)
	if err != nil {
		return Config{}, fmt.Errorf("parse GOOGLE_RECAPTCHA_MIN_SCORE: %w", err)
	}
	if googleRecaptchaMinScore < 0 || googleRecaptchaMinScore > 1 {
		return Config{}, fmt.Errorf("GOOGLE_RECAPTCHA_MIN_SCORE must be between 0 and 1")
	}

	sessionSecure, err := parseBoolWithDefault("SESSION_COOKIE_SECURE", appEnv == "prod")
	if err != nil {
		return Config{}, fmt.Errorf("parse SESSION_COOKIE_SECURE: %w", err)
	}

	sessionCookieSameSite, err := parseSameSiteWithDefault("SESSION_COOKIE_SAME_SITE", defaultSessionCookieSameSite)
	if err != nil {
		return Config{}, err
	}
	if sessionCookieSameSite == "none" && !sessionSecure {
		return Config{}, fmt.Errorf("SESSION_COOKIE_SECURE must be true when SESSION_COOKIE_SAME_SITE=none")
	}

	judgeTimeout, err := time.ParseDuration(valueOrDefault("JUDGE_TIMEOUT", defaultJudgeTimeout))
	if err != nil {
		return Config{}, fmt.Errorf("parse JUDGE_TIMEOUT: %w", err)
	}

	maxSourceCodeBytes, err := parseIntWithDefault("MAX_SOURCE_CODE_BYTES", defaultMaxSourceCodeBytes)
	if err != nil {
		return Config{}, fmt.Errorf("parse MAX_SOURCE_CODE_BYTES: %w", err)
	}
	if maxSourceCodeBytes <= 0 {
		return Config{}, fmt.Errorf("MAX_SOURCE_CODE_BYTES must be > 0")
	}

	maxBodyBytes, err := parseIntWithDefault("HTTP_MAX_BODY_BYTES", defaultMaxBodyBytes)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_MAX_BODY_BYTES: %w", err)
	}
	if maxBodyBytes <= 0 {
		return Config{}, fmt.Errorf("HTTP_MAX_BODY_BYTES must be > 0")
	}

	submissionRateLimit, err := parseIntWithDefault("SUBMISSION_RATE_LIMIT", defaultSubmissionRateLimit)
	if err != nil {
		return Config{}, fmt.Errorf("parse SUBMISSION_RATE_LIMIT: %w", err)
	}
	if submissionRateLimit <= 0 {
		return Config{}, fmt.Errorf("SUBMISSION_RATE_LIMIT must be > 0")
	}

	submissionRateWindow, err := time.ParseDuration(valueOrDefault("SUBMISSION_RATE_WINDOW", defaultSubmissionRateWindow))
	if err != nil {
		return Config{}, fmt.Errorf("parse SUBMISSION_RATE_WINDOW: %w", err)
	}
	if submissionRateWindow <= 0 {
		return Config{}, fmt.Errorf("SUBMISSION_RATE_WINDOW must be > 0")
	}

	matchDuration, err := time.ParseDuration(valueOrDefault("MATCH_DURATION", defaultMatchDuration))
	if err != nil {
		return Config{}, fmt.Errorf("parse MATCH_DURATION: %w", err)
	}
	if matchDuration <= 0 {
		return Config{}, fmt.Errorf("MATCH_DURATION must be > 0")
	}

	matchCleanupInterval, err := time.ParseDuration(valueOrDefault("MATCH_CLEANUP_INTERVAL", defaultMatchCleanupInterval))
	if err != nil {
		return Config{}, fmt.Errorf("parse MATCH_CLEANUP_INTERVAL: %w", err)
	}
	if matchCleanupInterval <= 0 {
		return Config{}, fmt.Errorf("MATCH_CLEANUP_INTERVAL must be > 0")
	}

	matchPendingTimeout, err := time.ParseDuration(valueOrDefault("MATCH_PENDING_TIMEOUT", defaultMatchPendingTimeout))
	if err != nil {
		return Config{}, fmt.Errorf("parse MATCH_PENDING_TIMEOUT: %w", err)
	}
	if matchPendingTimeout <= 0 {
		return Config{}, fmt.Errorf("MATCH_PENDING_TIMEOUT must be > 0")
	}

	backendBaseURL := normalizeBaseURL(valueOrDefault("BACKEND_PUBLIC_URL", defaultBackendBaseURL))
	judgeCallbackURL := normalizeBaseURL(valueOrDefault("JUDGE_CALLBACK_URL", backendBaseURL+"/internal/judge/callback"))
	judgeCallbackToken := strings.TrimSpace(os.Getenv("JUDGE_CALLBACK_TOKEN"))
	judgeAPIToken := strings.TrimSpace(os.Getenv("JUDGE_API_TOKEN"))
	resendAPIKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	emailFrom := strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	r2AccountID := strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID"))
	r2AccessKeyID := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
	r2SecretAccessKey := strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY"))
	r2Bucket := strings.TrimSpace(os.Getenv("R2_BUCKET"))
	r2PublicURL := normalizeBaseURL(os.Getenv("R2_PUBLIC_URL"))
	if anyNonEmpty(r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2Bucket, r2PublicURL) &&
		!allNonEmpty(r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2Bucket, r2PublicURL) {
		return Config{}, fmt.Errorf("R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, and R2_PUBLIC_URL must be set together")
	}
	if appEnv != "prod" {
		if judgeCallbackToken == "" {
			judgeCallbackToken = "dev-judge-callback-token"
		}
		if judgeAPIToken == "" {
			judgeAPIToken = "dev-judge-api-token"
		}
	}

	if appEnv == "prod" {
		if judgeCallbackToken == "" {
			return Config{}, fmt.Errorf("JUDGE_CALLBACK_TOKEN is required in prod")
		}
		if judgeAPIToken == "" {
			return Config{}, fmt.Errorf("JUDGE_API_TOKEN is required in prod")
		}
		if resendAPIKey == "" {
			return Config{}, fmt.Errorf("RESEND_API_KEY is required in prod")
		}
		if emailFrom == "" {
			return Config{}, fmt.Errorf("EMAIL_FROM is required in prod")
		}
	}

	return Config{
		AppEnv:                  appEnv,
		HTTPPort:                httpPort,
		DatabaseURL:             databaseURL,
		AllowedOrigins:          splitCSV(valueOrDefault("ALLOWED_ORIGINS", defaultAllowedOrigins)),
		FrontendBaseURL:         normalizeBaseURL(valueOrDefault("FRONTEND_PUBLIC_URL", defaultFrontendBaseURL)),
		BackendBaseURL:          backendBaseURL,
		SessionCookieName:       valueOrDefault("SESSION_COOKIE_NAME", defaultSessionCookieName),
		SessionCookieDomain:     strings.TrimSpace(os.Getenv("SESSION_COOKIE_DOMAIN")),
		SessionCookieSameSite:   sessionCookieSameSite,
		SessionTTL:              sessionTTL,
		SessionCleanupInterval:  sessionCleanupInterval,
		SessionSecure:           sessionSecure,
		OAuthStateCookie:        valueOrDefault("OAUTH_STATE_COOKIE_NAME", defaultOAuthStateCookie),
		OAuthStateTTL:           oauthStateTTL,
		GitHubClientID:          strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
		GitHubSecret:            strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
		GoogleClientID:          strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleSecret:            strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRecaptchaSecret:   strings.TrimSpace(os.Getenv("GOOGLE_RECAPTCHA_SECRET")),
		GoogleRecaptchaURL:      normalizeBaseURL(valueOrDefault("GOOGLE_RECAPTCHA_VERIFY_URL", defaultGoogleRecaptchaURL)),
		GoogleRecaptchaTimeout:  googleRecaptchaTimeout,
		GoogleRecaptchaMinScore: googleRecaptchaMinScore,
		GoogleRecaptchaAction:   strings.TrimSpace(os.Getenv("GOOGLE_RECAPTCHA_EXPECTED_ACTION")),
		ResendAPIKey:            resendAPIKey,
		EmailFrom:               emailFrom,
		R2AccountID:             r2AccountID,
		R2AccessKeyID:           r2AccessKeyID,
		R2SecretAccessKey:       r2SecretAccessKey,
		R2Bucket:                r2Bucket,
		R2PublicURL:             r2PublicURL,

		JudgeBaseURL:         normalizeBaseURL(valueOrDefault("JUDGE_BASE_URL", defaultJudgeBaseURL)),
		JudgeCallbackURL:     judgeCallbackURL,
		JudgeCallbackToken:   judgeCallbackToken,
		JudgeAPIToken:        judgeAPIToken,
		JudgeTimeout:         judgeTimeout,
		MaxSourceCodeBytes:   maxSourceCodeBytes,
		MaxBodyBytes:         maxBodyBytes,
		SubmissionRateLimit:  submissionRateLimit,
		SubmissionRateWindow: submissionRateWindow,
		MatchDuration:        matchDuration,
		MatchCleanupInterval: matchCleanupInterval,
		MatchPendingTimeout:  matchPendingTimeout,
	}, nil
}

func (c Config) ListenAddr() string {
	return "0.0.0.0:" + c.HTTPPort
}

func (c Config) R2Enabled() bool {
	return allNonEmpty(c.R2AccountID, c.R2AccessKeyID, c.R2SecretAccessKey, c.R2Bucket, c.R2PublicURL)
}

func valueOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		values = append(values, value)
	}

	return values
}

func normalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func anyNonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}

	return false
}

func allNonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}

	return true
}

func parseBoolWithDefault(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, err
	}

	return value, nil
}

func parseSameSiteWithDefault(key string, fallback string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(valueOrDefault(key, fallback)))
	switch value {
	case "lax", "strict", "none":
		return value, nil
	default:
		return "", fmt.Errorf("%s must be one of lax, strict, none", key)
	}
}

func parseIntWithDefault(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func parseFloatWithDefault(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}
