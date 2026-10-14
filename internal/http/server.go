package http

import (
	"context"
	"strings"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/managementhttp"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/metrics"
	"code-fight-backend/internal/realtime"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/tournament"
	"code-fight-backend/internal/users"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(
	cfg config.Config,
	authHandler *auth.Handler,
	usersHandler *users.Handler,
	tasksHandler *tasks.Handler,
	leaderboardHandler *leaderboard.Handler,
	matchesHandler *matches.Handler,
	realtimeHandler *realtime.Handler,
	submissionsHandler *submissions.Handler,
	backendMetrics *metrics.BackendMetrics,
	db *pgxpool.Pool,
	adminHandler *administration.Handler,
	tournamentHandler *tournament.Handler,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: jsonErrorHandler,
		BodyLimit:    cfg.MaxBodyBytes,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	if backendMetrics != nil {
		app.Use(backendMetrics.Middleware())
		app.Get("/metrics", backendMetrics.Handler())
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.AllowedOrigins, ","),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Judge-Token",
		AllowMethods:     "GET,POST,DELETE,PATCH,HEAD,OPTIONS",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"ok":      true,
			"service": "backend",
		})
	})
	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "backend",
		})
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":  "error",
				"service": "backend",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "backend",
		})
	})

	protectedGroup := func(path string, middlewares ...fiber.Handler) fiber.Router {
		group := app.Group(path)
		for _, middleware := range middlewares {
			group.Use(middleware)
		}
		group.Use(requireAllowedOrigin(cfg.AllowedOrigins))
		return group
	}

	internalGroup := func(path string, middlewares ...fiber.Handler) fiber.Router {
		group := app.Group(path)
		for _, middleware := range middlewares {
			group.Use(middleware)
		}
		return group
	}

	// Register before the shared /api origin guard as well as auth/role guards.
	// Only management routes opt into the structured error contract.
	app.Use("/api/admin", managementhttp.ErrorBoundary)
	app.Use("/api/organizer", managementhttp.ErrorBoundary)

	authGroup := protectedGroup("/auth", newAuthLimiter())
	authHandler.RegisterRoutes(authGroup)

	usersGroup := protectedGroup("/users")
	usersHandler.RegisterRoutes(usersGroup)

	tasksGroup := protectedGroup("/api")
	tasksHandler.RegisterRoutes(tasksGroup)
	leaderboardHandler.RegisterRoutes(tasksGroup)
	matchesHandler.RegisterPublicRoutes(tasksGroup)

	matchesGroup := protectedGroup(
		"/api/matches",
		authHandler.RequireAuth,
		newReplayCaptureLimiter(120, time.Minute),
	)
	matchesHandler.RegisterRoutes(matchesGroup)

	eventsGroup := protectedGroup("/api/events", authHandler.RequireAuth)
	realtimeHandler.RegisterRoutes(eventsGroup)

	adminGroup := protectedGroup("/api/admin", authHandler.RequireAuth, auth.RequireRoles(access.Admin))
	adminHandler.RegisterRoutes(adminGroup)
	organizerGroup := protectedGroup("/api/organizer", authHandler.RequireAuth, auth.RequireRoles(access.Organizer, access.Admin))
	tournamentHandler.RegisterRoutes(organizerGroup)

	submissionsAPIGroup := protectedGroup(
		"/api",
		authHandler.RequireAuth,
		newSubmissionLimiter(cfg.SubmissionRateLimit, cfg.SubmissionRateWindow),
	)
	submissionsInternalGroup := internalGroup("/internal")
	submissionsHandler.RegisterRoutes(submissionsAPIGroup, submissionsInternalGroup)
	matchesHandler.RegisterInternalRoutes(submissionsInternalGroup)

	return app
}
