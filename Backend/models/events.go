package models

type Event struct {
	ID                string `bson:"id,omitempty" json:"id"`
	Name              string `bson:"name" json:"name"`
	TotalTickets      int    `bson:"total_tickets" json:"total_tickets"`
	OrganizerUsername string `bson:"organizer_username" json:"organizer_username"`
}
