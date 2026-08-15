package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
)

func WSAuthRequired(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	tokenString := c.Query("token")
	secret := os.Getenv("JWT_SECRET")

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	claims := token.Claims.(jwt.MapClaims)
	username, ok := claims["username"].(string)
	if !ok || username == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "username missing from token")
	}

	c.Locals("username", username)
	return c.Next()
}
