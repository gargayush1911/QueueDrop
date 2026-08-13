package middleware

import "github.com/gofiber/fiber/v2"

func RequiredRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role").(string)

		for _, allowed := range allowedRoles {
			if role == allowed {
				return c.Next() // role matched proceed
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}
}
