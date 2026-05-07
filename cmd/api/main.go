package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/database"
	httpserver "code-fight-backend/internal/http"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/users"
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
	oauthClient := auth.NewOAuthClient(
		cfg.BackendBaseURL,
		cfg.GitHubClientID,
		cfg.GitHubSecret,
		cfg.GoogleClientID,
		cfg.GoogleSecret,
	)
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

	usersRepository := users.NewRepository(db)
	usersService := users.NewService(usersRepository)
	usersHandler := users.NewHandler(usersService, authService, cfg.SessionCookieName)

	judgeHTTPClient := judge.NewHTTPClient(cfg.JudgeBaseURL, cfg.JudgeTimeout, cfg.JudgeAPIToken)

	matchesRepo := matches.NewRepository(db)
	taskSelector := matches.NewDBTaskSelector(db)

	submissionsRepo := submissions.NewPGRepository(db)
	submissionsTaskRepo := submissions.NewPGTaskRepository(db)
	submissionsMatchReader := submissions.NewMatchReader(submissionsRepo)

	matchesService := matches.NewService(
		matchesRepo,
		taskSelector,
		submissionsMatchReader,
	)
	matchesHandler := matches.NewHandler(matchesService, cfg.MatchDuration)

	tasksRepo := tasks.NewRepository(db)
	tasksService := tasks.NewService(tasksRepo, judgeHTTPClient)
	tasksHandler := tasks.NewHandler(tasksService, authService, cfg.SessionCookieName)

	leaderboardRepo := leaderboard.NewRepository(db)
	leaderboardService := leaderboard.NewService(leaderboardRepo)
	leaderboardHandler := leaderboard.NewHandler(leaderboardService, authService, cfg.SessionCookieName)

	submissionsService := submissions.NewService(
		submissionsRepo,
		submissionsTaskRepo,
		matchesRepo,
		judgeHTTPClient,
		matchesService,
		cfg.JudgeCallbackURL,
		cfg.JudgeCallbackToken,
		cfg.MaxSourceCodeBytes,
	)
	submissionsHandler := submissions.NewHandler(
		submissionsService,
		cfg.JudgeCallbackToken,
	)

	cleanupCtx, stopSessionCleanup := context.WithCancel(context.Background())
	defer stopSessionCleanup()

	if cfg.SessionCleanupInterval > 0 {
		go startSessionCleanupLoop(cleanupCtx, logger, authService, cfg.SessionCleanupInterval)
	}
	if cfg.MatchCleanupInterval > 0 {
		go startMatchCleanupLoop(cleanupCtx, logger, matchesService, cfg.MatchCleanupInterval)
	}

	app := httpserver.New(
		cfg,
		authHandler,
		usersHandler,
		tasksHandler,
		leaderboardHandler,
		matchesHandler,
		submissionsHandler,
	)

	logger.Info("starting api", "addr", cfg.ListenAddr(), "app_env", cfg.AppEnv)

	if err := app.Listen(cfg.ListenAddr()); err != nil {
		logger.Error("listen", "error", err)
		os.Exit(1)
	}
}

func startSessionCleanupLoop(ctx context.Context, logger *slog.Logger, authService *auth.Service, interval time.Duration) {
	if err := authService.CleanupExpiredSessions(ctx); err != nil {
		logger.Error("cleanup expired sessions", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := authService.CleanupExpiredSessions(ctx); err != nil {
				logger.Error("cleanup expired sessions", "error", err)
			}
		}
	}
}

func startMatchCleanupLoop(ctx context.Context, logger *slog.Logger, matchesService *matches.Service, interval time.Duration) {
	if count, err := matchesService.FinishExpiredMatches(ctx, time.Now().UTC(), 50); err != nil {
		logger.Error("finish expired matches", "error", err)
	} else if count > 0 {
		logger.Info("finished expired matches", "count", count)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := matchesService.FinishExpiredMatches(ctx, time.Now().UTC(), 50)
			if err != nil {
				logger.Error("finish expired matches", "error", err)
				continue
			}
			if count > 0 {
				logger.Info("finished expired matches", "count", count)
			}
		}
	}
}
