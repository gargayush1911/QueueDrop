package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var MongoClient *mongo.Client
var UserCollection *mongo.Collection
var EventsCollection *mongo.Collection
var OrdersCollection *mongo.Collection

func InitMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017" // default URI if not set
	}

	clients, err := mongo.Connect(options.Client().ApplyURI(uri))
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

	_, err = UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	if err != nil {
		return err
	}

	_, err = OrdersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "event_id", Value: 1},
		},
	})
	if err != nil {
		return err
	}

	log.Println("MongoDB Connected!!")
	return nil
}
