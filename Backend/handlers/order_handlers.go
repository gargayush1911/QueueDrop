package handlers

import (
	"context"
	"time"

	"queuedrop/database"
	"queuedrop/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetMyOrders returns the authenticated user's own purchase orders,
// most recent first. This is how the frontend finds out whether a
// queue join ended up "confirmed" or "sold_out" (or is still pending,
// meaning the worker hasn't processed it yet).
func GetMyOrders(c *fiber.Ctx) error {
	username, ok := c.Locals("username").(string)

	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid authenticated user"},
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := database.OrdersCollection.Find(
		ctx,
		bson.M{"username": username},
		findOptions,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to fetch orders"},
		)
	}
	defer cursor.Close(ctx)

	orders := []models.Order{}

	if err := cursor.All(ctx, &orders); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": "failed to read orders"},
		)
	}

	return c.JSON(fiber.Map{"orders": orders})
}

// GetOrderStatus returns just the status for a single event, for the
// specific logged-in user — handy for polling right after joining a
// queue without having to filter the full order list client-side.
// Returns status "pending" if the worker hasn't processed the purchase
// yet (i.e. no order exists for this user+event combination).
func GetOrderStatus(c *fiber.Ctx) error {
	username, ok := c.Locals("username").(string)

	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			fiber.Map{"error": "invalid authenticated user"},
		)
	}

	eventIDParam := c.Params("id")

	objID, err := bson.ObjectIDFromHex(eventIDParam)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{"error": "invalid event ID"},
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order models.Order

	err = database.OrdersCollection.FindOne(
		ctx,
		bson.M{"username": username, "event_id": objID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&order)

	if err != nil {
		// No order yet means the worker hasn't processed this purchase
		// request yet — not an error, just "still pending".
		return c.JSON(fiber.Map{"status": "pending"})
	}

	return c.JSON(fiber.Map{"status": order.Status})
}
