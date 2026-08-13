package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := database.EventsCollection.InsertOne(ctx, event)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	eventID := result.InsertedID.(bson.ObjectID)
	stockKey := fmt.Sprintf("event:%s:stock", eventID.Hex())
	cache.RedisClient.Set(cache.Ctx, stockKey, event.TotalTickets, 0)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"inserted_id": eventID.Hex()})
}

func GetEvents(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := database.EventsCollection.Find(ctx, bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	defer cursor.Close(ctx)

	events := []models.Event{}
	if err := cursor.All(ctx, &events); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(events)
}

func UpdateEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	username := c.Locals("username").(string)
	role := c.Locals("role").(string)

	objID, _ := bson.ObjectIDFromHex(eventID)

	var event models.Event
	err := database.EventsCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&event)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	// admin can edit anything; organizer can only edit their OWN event
	if role != "admin" && event.OrganizerUsername != username {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you don't own this event"})
	}

	var update models.Event
	c.BodyParser(&update)

	database.EventsCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"name": update.Name, "total_tickets": update.TotalTickets}},
	)

	return c.JSON(fiber.Map{"message": "event updated"})
}

func JoinQueue(c *fiber.Ctx) error {
	eventID := c.Params("id")
	username := c.Locals("username").(string)

	request := PurchaseRequest{
		EventID:  eventID,
		Username: username,
	}

	body, _ := json.Marshal(request)

	err := queue.Channel.Publish(
		"",               // exchange (empty = default)
		"purchase_queue", // routing key = queue name
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		log.Println("failed to publish to queue:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to join queue"})
	}

	return c.JSON(fiber.Map{"message": "you've joined the queue", "event_id": eventID})

}
