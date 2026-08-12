package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var MongoClient *mongo.Client
var UserCollection *mongo.Collection
var EventsCollection *mongo.Collection
var OrdersCollection *mongo.Collection

func InitMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clients, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27018"))
	if err != nil {
		log.Fatal("MongoDB connection failed: ", err)
	}

	if err := clients.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed")
	}

	MongoClient = clients
	db := clients.Database("queuedrop")

	UserCollection = db.Collection("users")
	EventsCollection = db.Collection("events")
	OrdersCollection = db.Collection("orders")

	log.Println("MongoDB Connected!!")
}
