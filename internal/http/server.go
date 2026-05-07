package http

import (
	"strings"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/users"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func New(
	cfg config.Config,
	authHandler *auth.Handler,
	usersHandler *users.Handler,
	tasksHandler *tasks.Handler,
	leaderboardHandler *leaderboard.Handler,
	matchesHandler *matches.Handler,
	submissionsHandler *submissions.Handler,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: jsonErrorHandler,
		BodyLimit:    cfg.MaxBodyBytes,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.AllowedOrigins, ","),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Judge-Token",
		AllowMethods:     "GET,POST,PATCH,HEAD,OPTIONS",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"ok":      true,
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

	authGroup := protectedGroup("/auth", newAuthLimiter())
	authHandler.RegisterRoutes(authGroup)

	usersGroup := protectedGroup("/users")
	usersHandler.RegisterRoutes(usersGroup)

	tasksGroup := protectedGroup("/api")
	tasksHandler.RegisterRoutes(tasksGroup)
	leaderboardHandler.RegisterRoutes(tasksGroup)

	matchesGroup := protectedGroup("/matches")
	matchesHandler.RegisterRoutes(matchesGroup)

	submissionsAPIGroup := protectedGroup(
		"/api",
		authHandler.RequireAuth,
		newSubmissionLimiter(cfg.SubmissionRateLimit, cfg.SubmissionRateWindow),
	)
	submissionsInternalGroup := internalGroup("/internal")
	submissionsHandler.RegisterRoutes(submissionsAPIGroup, submissionsInternalGroup)

	return app
}
