package main

import (
	"log"
	"queuedrop/cache"
	"queuedrop/database"
	"queuedrop/handlers"
	"queuedrop/middleware"
	"queuedrop/queue"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	database.InitMongoDB()
	cache.InitRedis()
	queue.InitRabbitMQ()
	queue.StartWorker()

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

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
	app.Get("/ws/notifications", middleware.WSAuthRequired, websocket.New(handlers.HandleNotifications))

	log.Fatal(app.Listen(":8080"))
}
