package http

import "github.com/gofiber/fiber/v2"

func jsonErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	if fiberError, ok := err.(*fiber.Error); ok {
		code = fiberError.Code
		message = fiberError.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
