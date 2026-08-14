package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Users struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string        `bson:"username" json:"username"`
	Password string        `bson:"password" json:"-"`
	Role     string        `bson:"role" json:"role"` // "user", "organizer", or "admin"
}
