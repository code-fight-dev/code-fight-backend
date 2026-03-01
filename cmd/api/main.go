package main

import (
	"context"
	"log/slog"
	"os"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/database"
	httpserver "code-fight-backend/internal/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}

	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository, cfg.SessionTTL)
	oauthClient := auth.NewOAuthClient(cfg.BackendBaseURL, cfg.GitHubClientID, cfg.GitHubSecret, cfg.GoogleClientID, cfg.GoogleSecret)
	authHandler := auth.NewHandler(
		authService,
		oauthClient,
		cfg.FrontendBaseURL,
		cfg.SessionCookieName,
		cfg.OAuthStateCookie,
		cfg.SessionTTL,
		cfg.OAuthStateTTL,
		cfg.SessionSecure,
	)

	app := httpserver.New(cfg, authHandler)

	logger.Info("starting api", "addr", cfg.ListenAddr(), "app_env", cfg.AppEnv)

	if err := app.Listen(cfg.ListenAddr()); err != nil {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
}
