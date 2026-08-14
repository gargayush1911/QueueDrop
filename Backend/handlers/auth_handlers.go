package handlers

import (
	"context"
	"os"
	"strings"
	"time"

	"queuedrop/database"
	"queuedrop/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Register(c *fiber.Ctx) error {
	var input RegisterRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if input.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "username is required"},
		)
	}

	if len(input.Username) < 3 || len(input.Username) > 30 {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "username must be between 3 and 30 characters"},
		)
	}

	if len(input.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "password must be at least 6 characters"},
		)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not hash password"})
	}
	user := models.Users{
		Username: input.Username,
		Password: string(hashedPassword),
		Role:     "user", // default role
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.UserCollection.InsertOne(ctx, user)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "username already taken"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"username": input.Username, "role": user.Role})
}

func Login(c *fiber.Ctx) error {
	var loginInput LoginRequest
	if err := c.BodyParser(&loginInput); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	loginInput.Username = strings.TrimSpace(loginInput.Username)
	if loginInput.Username == "" || loginInput.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "username and password are required"},
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.Users
	err := database.UserCollection.FindOne(ctx, bson.M{"username": loginInput.Username}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginInput.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid username or password"})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}
