package main

import (
	"log"
	"queuedrop/database"
	"queuedrop/handlers"

	"github.com/gofiber/fiber/v2"
)

func main() {
	database.InitMongoDB()

	app := fiber.New()

	app.Post("/api/register", handlers.Register)

	log.Fatal(app.Listen(":8080"))
}
