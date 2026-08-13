package main

import (
	"log"
	"queuedrop/cache"
	"queuedrop/database"
	"queuedrop/handlers"
	"queuedrop/middleware"
	"queuedrop/queue"

	"github.com/gofiber/fiber/v2"
)

func main() {
	database.InitMongoDB()
	cache.InitRedis()
	queue.InitRabbitMQ()
	queue.StartWorker()

	app := fiber.New()

	app.Post("/api/register", handlers.Register)
	app.Post("/api/login", handlers.Login)

	app.Post("/api/events", middleware.AuthRequired,
		middleware.RequiredRole("organizer", "admin"), handlers.CreateEvent)

	app.Get("/api/events", handlers.GetEvents)

	app.Put("/api/events/:id",
		middleware.AuthRequired,
		middleware.RequiredRole("organizer", "admin"),
		handlers.UpdateEvent)

	app.Post("/api/events/:id/queue", middleware.AuthRequired, handlers.JoinQueue)

	log.Fatal(app.Listen(":8080"))
}
