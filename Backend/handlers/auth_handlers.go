package handlers

import (
	"context"
	"os"
	"time"

	"queuedrop/database"
	"queuedrop/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *fiber.Ctx) error {
	var input models.Users
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not hash password"})
	}
	input.Password = string(hashedPassword)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = database.UserCollection.InsertOne(ctx, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username already taken"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"username": input.Username})
}

func Login(c *fiber.Ctx) error {
	var loginInput models.Users
	if err := c.BodyParser(&loginInput); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}
