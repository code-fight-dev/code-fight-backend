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
	defaultSessionTTL             = "720h"
	defaultSessionCleanupInterval = "1h"
	defaultAllowedOrigins         = "http://localhost:3000"
	defaultFrontendBaseURL        = "http://localhost:3000"
	defaultBackendBaseURL         = "http://localhost:8080"
	defaultOAuthStateCookie       = "codefight_oauth_state"
	defaultOAuthStateTTL          = "10m"
)

type Config struct {
	AppEnv                 string
	HTTPPort               string
	DatabaseURL            string
	AllowedOrigins         []string
	FrontendBaseURL        string
	BackendBaseURL         string
	SessionCookieName      string
	SessionTTL             time.Duration
	SessionCleanupInterval time.Duration
	SessionSecure          bool
	OAuthStateCookie       string
	OAuthStateTTL          time.Duration
	GitHubClientID         string
	GitHubSecret           string
	GoogleClientID         string
	GoogleSecret           string
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

	sessionSecure, err := parseBoolWithDefault("SESSION_COOKIE_SECURE", appEnv == "prod")
	if err != nil {
		return Config{}, fmt.Errorf("parse SESSION_COOKIE_SECURE: %w", err)
	}

	return Config{
		AppEnv:                 appEnv,
		HTTPPort:               httpPort,
		DatabaseURL:            databaseURL,
		AllowedOrigins:         splitCSV(valueOrDefault("ALLOWED_ORIGINS", defaultAllowedOrigins)),
		FrontendBaseURL:        normalizeBaseURL(valueOrDefault("FRONTEND_PUBLIC_URL", defaultFrontendBaseURL)),
		BackendBaseURL:         normalizeBaseURL(valueOrDefault("BACKEND_PUBLIC_URL", defaultBackendBaseURL)),
		SessionCookieName:      valueOrDefault("SESSION_COOKIE_NAME", defaultSessionCookieName),
		SessionTTL:             sessionTTL,
		SessionCleanupInterval: sessionCleanupInterval,
		SessionSecure:          sessionSecure,
		OAuthStateCookie:       valueOrDefault("OAUTH_STATE_COOKIE_NAME", defaultOAuthStateCookie),
		OAuthStateTTL:          oauthStateTTL,
		GitHubClientID:         strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
		GitHubSecret:           strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
		GoogleClientID:         strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleSecret:           strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
	}, nil
}

func (c Config) ListenAddr() string {
	return "0.0.0.0:" + c.HTTPPort
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
