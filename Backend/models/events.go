package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Event struct {
	ID                bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name              string        `bson:"name" json:"name"`
	TotalTickets      int           `bson:"total_tickets" json:"total_tickets"`
	OrganizerUsername string        `bson:"organizer_username" json:"organizer_username"`
}

type UpdateEventRequest struct {
	Name         *string `json:"name"`
	TotalTickets *int    `json:"total_tickets"`
}
