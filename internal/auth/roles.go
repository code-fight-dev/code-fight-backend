package auth

import (
	"code-fight-backend/internal/access"

	"github.com/gofiber/fiber/v2"
)

func RequireRoles(roles ...access.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(PublicUser)
		if !ok {
			return fiber.ErrUnauthorized
		}
		for _, role := range roles {
			if role.Valid() && user.Role == role {
				return c.Next()
			}
		}
		return fiber.ErrForbidden
	}
}
