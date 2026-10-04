package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/database"
	httpserver "code-fight-backend/internal/http"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/mail"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/metrics"
	"code-fight-backend/internal/objectstorage"
	"code-fight-backend/internal/realtime"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/tournament"
	"code-fight-backend/internal/users"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	if err := run(ctx, logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

var (
	loadConfig      = config.Load
	connectDatabase = database.Connect
	closeDatabase   = func(db *pgxpool.Pool) {
		db.Close()
	}
	migrateDatabase = func(ctx context.Context, db *pgxpool.Pool) error {
		return database.Migrate(ctx, db)
	}
	newResendSender = mail.NewResendSender
	newR2Client     = objectstorage.NewR2Client
	newHTTPApp      = httpserver.New
	listenHTTPApp   = func(app *fiber.App, addr string) error {
		return app.Listen(addr)
	}
)

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := connectDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer closeDatabase(db)

	if err := migrateDatabase(ctx, db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	authRepository := auth.NewRepository(db)
	passwordResetSender := auth.PasswordResetMailer(mail.NewLogSender(logger))
	if cfg.ResendAPIKey != "" && cfg.EmailFrom != "" {
		resendSender, err := newResendSender(cfg.ResendAPIKey, cfg.EmailFrom)
		if err != nil {
			return fmt.Errorf("create resend sender: %w", err)
		}
		passwordResetSender = resendSender
	}

	authService := auth.NewService(authRepository, cfg.SessionTTL, passwordResetSender)
	oauthClient := auth.NewOAuthClient(
		cfg.BackendBaseURL,
		cfg.GitHubClientID,
		cfg.GitHubSecret,
		cfg.GoogleClientID,
		cfg.GoogleSecret,
	)
	var captchaVerifier auth.CaptchaVerifier
	if cfg.GoogleRecaptchaSecret != "" {
		captchaVerifier = auth.NewGoogleRecaptchaVerifier(
			cfg.GoogleRecaptchaSecret,
			cfg.GoogleRecaptchaURL,
			cfg.GoogleRecaptchaTimeout,
			cfg.GoogleRecaptchaMinScore,
			cfg.GoogleRecaptchaAction,
		)
	}
	authHandler := auth.NewHandler(
		authService,
		oauthClient,
		captchaVerifier,
		cfg.FrontendBaseURL,
		cfg.SessionCookieName,
		cfg.SessionCookieDomain,
		cfg.SessionCookieSameSite,
		cfg.OAuthStateCookie,
		cfg.SessionTTL,
		cfg.OAuthStateTTL,
		cfg.SessionSecure,
	)

	var avatarStorage users.AvatarStorage
	if cfg.R2Enabled() {
		r2Client, err := newR2Client(ctx, objectstorage.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			Bucket:          cfg.R2Bucket,
			PublicURL:       cfg.R2PublicURL,
		})
		if err != nil {
			return fmt.Errorf("create r2 client: %w", err)
		}
		avatarStorage = r2Client
	}

	backendMetrics := metrics.NewBackendMetrics()

	usersRepository := users.NewRepository(db)
	usersService := users.NewService(usersRepository, avatarStorage, backendMetrics)
	usersHandler := users.NewHandler(usersService, authService, cfg.SessionCookieName)

	judgeHTTPClient := judge.NewHTTPClient(cfg.JudgeBaseURL, cfg.JudgeTimeout, cfg.JudgeAPIToken)

	matchesRepo := matches.NewRepository(db)
	taskSelector := matches.NewDBTaskSelector(db)
	realtimeBroker := realtime.NewBroker()
	realtimeHandler := realtime.NewHandler(realtimeBroker)

	submissionsRepo := submissions.NewPGRepository(db)
	submissionsTaskRepo := submissions.NewPGTaskRepository(db)
	submissionsMatchReader := submissions.NewMatchReader(submissionsRepo)

	matchesService := matches.NewService(
		matchesRepo,
		taskSelector,
		submissionsMatchReader,
	)
	matchesService.SetNotifier(realtimeBroker)
	matchesService.SetMetrics(backendMetrics)
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
		backendMetrics,
	)
	submissionsService.SetNotifier(realtimeBroker)
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
		go startMatchCleanupLoop(cleanupCtx, logger, matchesService, cfg.MatchCleanupInterval, cfg.MatchPendingTimeout)
	}

	app := newHTTPApp(
		cfg,
		authHandler,
		usersHandler,
		tasksHandler,
		leaderboardHandler,
		matchesHandler,
		realtimeHandler,
		submissionsHandler,
		backendMetrics,
		db,
		administration.NewHandler(administration.NewService(db)),
		tournament.NewHandler(tournament.NewService(db)),
	)

	logger.Info("starting api", "addr", cfg.ListenAddr(), "app_env", cfg.AppEnv)

	if err := listenHTTPApp(app, cfg.ListenAddr()); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

type sessionCleanupService interface {
	CleanupExpiredSessions(ctx context.Context) error
}

type matchCleanupService interface {
	FinishExpiredMatches(ctx context.Context, now time.Time, limit int) (int, error)
	CancelExpiredPendingMatches(ctx context.Context, now time.Time, pendingTimeout time.Duration, limit int) (int, error)
}

func startSessionCleanupLoop(ctx context.Context, logger *slog.Logger, authService sessionCleanupService, interval time.Duration) {
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

func startMatchCleanupLoop(
	ctx context.Context,
	logger *slog.Logger,
	matchesService matchCleanupService,
	interval time.Duration,
	pendingTimeout time.Duration,
) {
	if count, err := matchesService.FinishExpiredMatches(ctx, time.Now().UTC(), 50); err != nil {
		logger.Error("finish expired matches", "error", err)
	} else if count > 0 {
		logger.Info("finished expired matches", "count", count)
	}
	if count, err := matchesService.CancelExpiredPendingMatches(ctx, time.Now().UTC(), pendingTimeout, 50); err != nil {
		logger.Error("cancel expired pending matches", "error", err)
	} else if count > 0 {
		logger.Info("cancelled expired pending matches", "count", count)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()

			count, err := matchesService.FinishExpiredMatches(ctx, now, 50)
			if err != nil {
				logger.Error("finish expired matches", "error", err)
			} else if count > 0 {
				logger.Info("finished expired matches", "count", count)
			}

			count, err = matchesService.CancelExpiredPendingMatches(ctx, now, pendingTimeout, 50)
			if err != nil {
				logger.Error("cancel expired pending matches", "error", err)
			} else if count > 0 {
				logger.Info("cancelled expired pending matches", "count", count)
			}
		}
	}
}
