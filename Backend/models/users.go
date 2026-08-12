package models

type Users struct {
	ID       string `bson:"id,omitempty" json:"id"`
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password,omitempty"`
}
