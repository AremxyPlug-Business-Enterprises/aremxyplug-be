package models

import "time"

type Transaction struct {
	Product     string    `bson:"product" json:"product"`
	Description string    `bson:"description" json:"description"`
	OrderID     int       `bson:"order_id" json:"order_id"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	Status      string    `bson:"status" json:"status"`
}
