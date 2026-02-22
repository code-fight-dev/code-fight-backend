package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"ok":      true,
			"service": "backend",
		})
	})

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	_ = app.Listen("0.0.0.0:" + port)
}