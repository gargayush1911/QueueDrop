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

const reserveTicketScript = `
local stock = redis.call("GET", KEYS[1])
if not stock then 
return -2
end

if tonumber(stock) <= 0 then
return -1
end

return redis.call("DECR", KEYS[1])
`

func StartWorker() {
	if Channel == nil {
		log.Fatal("RabbitMQ channel is not initialized")
	}

	msgs, err := Channel.Consume(
		PurchaseQueue,
		"",    // consumer tag (auto-generated)
		false,  // auto-ack — we'll discuss the tradeoff below
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
			if err:= processPurchase(msg.Body);err!=nil{
				log.Println("purchase processing failed: ",err)
				continue
			}

			if err:= msg.Ack(false); err != nil {
				log.Println("failed to ack message: ", err)
			}
		}
	}()
	log.Println("worker started, listening for purchase request")
}

func processPurchase(body []byte) error {
	log.Println("worker received message:", string(body))

	var req Purchaserequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("invalid purchase message : %w",err)
	}

	if req.EventID == "" || req.Username == "" {
		return fmt.Errorf("invalid purchase message: missing event_id or username")
	}

	eventID , err:= bson.ObjectIDFromHex(req.EventID)
	if err != nil {
		return fmt.Errorf("invalid event_id: %w", err)
	}

	ctx,cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event models.Event
	if err := database.EventsCollection.FindOne(ctx, bson.M{"_id": eventID}).Decode(&event); err != nil {
		return fmt.Errorf("event not found: %w", err)
	}


	stockkey := fmt.Sprintf("event:%s:stock", req.EventID)

	result,err:= cache.RedisClient.Eval(cache.Ctx,reserveTicketScript,[]string{stockkey},).Int64()
	if err != nil {
		log.Println("redis eval failed:", err)
		return err
	}

	if result==-1{
		order := models.Order{
			EventID:   req.EventID,
			Username:  req.Username,
			Status:    "sold_out",
			CreatedAt: time.Now(),
		}
	}

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
