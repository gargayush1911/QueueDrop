package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or Invalid authorization header"})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or Expired token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}
	username, ok := claims["username"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "username missing from the token"})
	}
	role, ok := claims["role"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "role missing from the token"})
	}
	c.Locals("username", username)
	c.Locals("role", role)
	return c.Next()
}
