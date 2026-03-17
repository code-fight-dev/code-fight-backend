package http

import (
	"strings"

	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/matches"
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
	matchesHandler *matches.Handler,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: jsonErrorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.AllowedOrigins, ","),
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowMethods:     "GET,POST,HEAD,OPTIONS",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"ok":      true,
			"service": "backend",
		})
	})

	authGroup := app.Group("/auth")
	authGroup.Use(newAuthLimiter())
	authGroup.Use(requireAllowedOrigin(cfg.AllowedOrigins))
	authHandler.RegisterRoutes(authGroup)

	usersGroup := app.Group("/users")
	usersGroup.Use(requireAllowedOrigin(cfg.AllowedOrigins))
	usersHandler.RegisterRoutes(usersGroup)

	matchesGroup := app.Group("/matches")
	matchesGroup.Use(requireAllowedOrigin(cfg.AllowedOrigins))
	matchesHandler.RegisterRoutes(matchesGroup)

	return app
}