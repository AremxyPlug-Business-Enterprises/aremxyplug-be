package telcom

import "time"

type DataResult struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"Status" bson:"status"`
	Network                string    `json:"network" bson:"network"`                 // "MTN"
	NetworkProduct         string    `json:"network_product" bson:"network_product"` // "MTN SME"
	PlanName               string    `json:"plan_name" bson:"plan_name"`             // "MTN SME 10GB (₦100) ~ 1 MONTH"
	PhoneNumber            string    `json:"phone_number" bson:"phone_number"`
	RecipientName          string    `json:"recipient_name" bson:"recipient_name"`
	Plan_Amount            string    `json:"amount" bson:"amount"`
	FullName               string    `json:"full_name" bson:"full_name"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`         // "Data Top-up"
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"` // "MTN SME"
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber        string    `json:"reference_number" bson:"reference_number"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
	ApiID                  int       `bson:"apiID"`
}
