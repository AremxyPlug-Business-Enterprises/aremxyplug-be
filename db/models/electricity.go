package models

import "time"

type ElectricResult struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`
	DiscoType              string    `json:"disco_type" bson:"DiscoType"`
	MeterType              string    `json:"meter_type" bson:"meter_type"` // Prepaid
	VerifiedName           string    `json:"verified_name" bson:"name"`
	MeterNumber            string    `json:"meter_number" bson:"meter_number"`
	Phone                  string    `json:"phone" bson:"phone"`
	Email                  string    `json:"email" bson:"email"`
	Amount                 string    `json:"amount" bson:"amount"`
	FullName               string    `json:"full_name" bson:"full_name"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"` // append serviceID and variation code.
	BillGenerated          string    `json:"bill_generated" bson:"bill_generated"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNuber         string    `json:"reference_number" bson:"reference_number"`
	RequestID              string    `bson:"request_ID"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
}
