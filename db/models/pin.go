package models

import "time"

type UserPin struct {
	UserID    string    `json:"user_id" bson:"user_id"`
	Pin       string    `json:"pin" bson:"pin"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}
