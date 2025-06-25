package telcom

import "time"

type AirtimeInfo struct {
	UserID    string
	Network   string `json:"network"`
	Amount    string `json:"amount"`
	Phone_no  string `json:"mobileno"`
	Recipient string `json:"recipient,omitempty"`
	FullName  string
}

type AirtimeApiResponse struct {
	Success_Response string  `json:"success"`
	Message          string  `json:"message"`
	Network          string  `json:"network"`
	Phone_no         string  `json:"mobileno"`
	Amount           int     `json:"airtimeamount"`
	Charged          float64 `json:"amountcharged"`
	Status           string  `json:"status"`
	Date             string  `json:"transaction_date"`
	Reference        string  `json:"reference_no"`
}

type AirtimeResponse struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`
	Network                string    `json:"network" bson:"network"`
	NetworkProduct         string    `json:"network_product" bson:"network_product"`
	Amount                 string    `json:"amount" bson:"amount"`
	Phone_no               string    `json:"phone_no" bson:"phone_no"`
	FullName               string    `json:"full_name" bson:"full_name"`
	Product                string    `json:"product" bson:"product"`
	RecipientName          string    `json:"recipient_name,omitempty" bson:"recipient_name,omitempty"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber        string    `json:"reference_number" bson:"reference_number"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
}
