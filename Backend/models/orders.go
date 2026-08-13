package models

import "time"

type Order struct {
	ID        string    `bson:"id,omitempty" json:"id"`
	EventID   string    `bson:"event_id" json:"event_id"`
	Username  string    `bson:"username" json:"username"`
	Status    string    `bson:"status" json:"status"` //confirmed or sold-out
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
