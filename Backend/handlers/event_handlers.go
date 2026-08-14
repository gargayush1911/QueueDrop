package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"queuedrop/cache"
	"queuedrop/database"
	"queuedrop/models"
	"queuedrop/queue"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/v2/bson"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PurchaseRequest struct {
	EventID  string `json:"event_id"`
	Username string `json:"username"`
}

func CreateEvent(c *fiber.Ctx) error {
	var event models.Event

	if err := c.BodyParser(&event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid request body"},
		)
	}

	event.Name = strings.TrimSpace(event.Name)

	if event.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "event name is required"},
		)
	}

	if event.TotalTickets <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "total tickets must be greater than zero"},
		)
	}

	username, ok := c.Locals("username").(string)

	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid authenticated user"},
		)
	}

	// Never trust organizer_username from the request body.
	event.OrganizerUsername = username

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := database.EventsCollection.InsertOne(ctx, event)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to create event"},
		)
	}

	eventID, ok := result.InsertedID.(bson.ObjectID)

	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "invalid event ID generated"},
		)
	}

	stockKey := fmt.Sprintf(
		"event:%s:stock",
		eventID.Hex(),
	)

	err = cache.RedisClient.Set(
		cache.Ctx,
		stockKey,
		event.TotalTickets,
		0,
	).Err()

	if err != nil {
		// Prevent an event existing without its inventory.
		_, _ = database.EventsCollection.DeleteOne(
			ctx,
			bson.M{"_id": eventID},
		)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to initialize event inventory"},
		)
	}

	return c.Status(fiber.StatusCreated).JSON(
		fiber.Map{
			"message":     "event created successfully",
			"inserted_id": eventID.Hex(),
		},
	)
}

func GetEvents(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := database.EventsCollection.Find(ctx, bson.M{})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to fetch events"},
		)
	}

	defer cursor.Close(ctx)

	events := []models.Event{}

	if err := cursor.All(ctx, &events); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to decode events"},
		)
	}

	return c.JSON(events)
}

func UpdateEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")

	objID, err := bson.ObjectIDFromHex(eventID)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid event ID"},
		)
	}

	username, ok := c.Locals("username").(string)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid authenticated user"},
		)
	}

	role, ok := c.Locals("role").(string)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid role"},
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event models.Event

	err = database.EventsCollection.FindOne(
		ctx,
		bson.M{"_id": objID},
	).Decode(&event)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": "event not found"},
		)
	}

	// Admin can edit anything.
	// Organizer can only edit their own event.
	if role != "admin" && event.OrganizerUsername != username {
		return c.Status(fiber.StatusForbidden).JSON(
			fiber.Map{"error": "you don't own this event"},
		)
	}

	var update models.UpdateEventRequest

	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid request body"},
		)
	}

	updateFields := bson.M{}

	if update.Name != nil {
		name := strings.TrimSpace(*update.Name)

		if name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "event name cannot be empty"},
			)
		}

		updateFields["name"] = name
	}

	if update.TotalTickets != nil {
		if *update.TotalTickets <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{"error": "total tickets must be greater than zero"},
			)
		}

		stockKey := fmt.Sprintf(
			"event:%s:stock",
			eventID,
		)

		currentStock, err := cache.RedisClient.Get(
			cache.Ctx,
			stockKey,
		).Int()

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				fiber.Map{"error": "failed to read ticket inventory"},
			)
		}

		soldTickets := event.TotalTickets - currentStock

		if *update.TotalTickets < soldTickets {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{
					"error": "new ticket count cannot be lower than tickets already sold",
				},
			)
		}

		newStock := *update.TotalTickets - soldTickets

		if err := cache.RedisClient.Set(
			cache.Ctx,
			stockKey,
			newStock,
			0,
		).Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				fiber.Map{"error": "failed to update ticket inventory"},
			)
		}

		updateFields["total_tickets"] = *update.TotalTickets
	}

	if len(updateFields) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "no fields to update"},
		)
	}

	_, err = database.EventsCollection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to update event"},
		)
	}

	return c.JSON(
		fiber.Map{"message": "event updated successfully"},
	)
}

func JoinQueue(c *fiber.Ctx) error {
	eventID := c.Params("id")

	// Validate event ID before publishing.
	objID, err := bson.ObjectIDFromHex(eventID)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid event ID"},
		)
	}

	username, ok := c.Locals("username").(string)

	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid authenticated user"},
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify event exists.
	var event models.Event

	err = database.EventsCollection.FindOne(
		ctx,
		bson.M{"_id": objID},
	).Decode(&event)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": "event not found"},
		)
	}

	request := PurchaseRequest{
		EventID:  eventID,
		Username: username,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to create queue request"},
		)
	}

	if queue.Channel == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			fiber.Map{"error": "queue service unavailable"},
		)
	}

	err = queue.Channel.Publish(
		"",
		queue.PurchaseQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)

	if err != nil {
		log.Println("failed to publish purchase:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to join queue"},
		)
	}
	return c.JSON(fiber.Map{
		"message":  "you've joined the queue",
		"event_id": eventID,
		"username": username})
}
