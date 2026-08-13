package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"queuedrop/database"
	"queuedrop/models"
	"time"

	"queuedrop/cache"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Purchaserequest struct {
	EventID  string `json:"event_id"`
	Username string `json:"username"`
}

func StartWorker() {
	msgs, err := Channel.Consume(
		"purchase_queue",
		"",    // consumer tag (auto-generated)
		true,  // auto-ack — we'll discuss the tradeoff below
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("failed to start consuming: ", err)
	}

	go func() {
		for msg := range msgs { // blocks here, processes one message at a time as they arrive
			processPurchase(msg.Body)
		}
	}()
	log.Println("worker started, listening for purchase request")
}

func processPurchase(body []byte) {
	log.Println("worker received message:", string(body)) // ADD THIS
	var req Purchaserequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Println("failed to unmarshal message:", err) // ADD THIS
		return
	}
	log.Println("parsed request:", req) // ADD THIS

	stockkey := fmt.Sprintf("event:%s:stock", req.EventID)

	// atomic decrement — this is the safety guarantee against overselling
	remaining, err := cache.RedisClient.Decr(cache.Ctx, stockkey).Result()
	if err != nil {
		log.Println("redis decr failed:", err)
		return
	}
	log.Println("stock after decrement:", remaining) // ADD THIS

	status := "confirmed"
	if remaining < 0 {
		status = "sold_out"
		cache.RedisClient.Incr(cache.Ctx, stockkey) // give the "slot" back since nobody actually got it
	}

	objID, _ := bson.ObjectIDFromHex(req.EventID)
	order := models.Order{
		EventID:   req.EventID,
		Username:  req.Username,
		Status:    status,
		CreatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	database.OrdersCollection.InsertOne(ctx, order)
	_ = objID

	log.Printf("processed purchase: user=%s event=%s status=%s (remaining=%d)\n", req.Username, req.EventID, status, remaining)
}
