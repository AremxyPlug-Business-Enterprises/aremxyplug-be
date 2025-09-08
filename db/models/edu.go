package models

import "time"

type EduResponse struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`
	Exam_Type              string    `json:"exam_type" bson:"exam_type"`
	Quantity               int       `json:"quantity" bson:"quantity"`
	PhoneNumber            string    `json:"phone_number" bson:"phone_number"`
	Email                  string    `json:"email" bson:"email"`
	Amount                 float64   `json:"amount" bson:"amount"`
	FullName               string    `json:"full_name" bson:"full_name"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	Pin_Generated          []string  `json:"pins_generated" bson:"pins_generated"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber        string    `json:"reference_no" bson:"reference_no"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
	TXN                    string    `bson:"txn"`
}

type EduRecord struct {
	ID     int
	Amount string
	Name   string
}
