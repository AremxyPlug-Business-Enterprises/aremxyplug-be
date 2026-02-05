package telcom

import "time"

type AirtimeResponse struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`
	Network                string    `json:"network" bson:"network"`
	NetworkProduct         string    `json:"network_product" bson:"network_product"`
	Amount                 string    `json:"amount" bson:"amount"`
	Phone_no               string    `json:"phone_no" bson:"phone_no"`
	FullName               string    `json:"full_name" bson:"full_name"`
	RecipientName          string    `json:"recipient_name,omitempty" bson:"recipient_name,omitempty"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber        string    `json:"reference_number" bson:"reference_number"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
	UserReference          string    `json:"user_reference" bson:"user_reference"`
	Discount_perecent      string    `json:"discount_percentage" bson:"discount_percentage"`
	Discount_amount        string    `json:"discount_amount" bson:"discount_amount"`
}
