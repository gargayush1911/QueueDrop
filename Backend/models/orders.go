package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Order struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID   bson.ObjectID `bson:"event_id" json:"event_id"`
	Username  string        `bson:"username" json:"username"`
	Status    string        `bson:"status" json:"status"` //confirmed or sold-out
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}
