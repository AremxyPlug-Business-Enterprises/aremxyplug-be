package models

import "time"

type TV_Result struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`
	DecoderType            string    `json:"decoder_type" bson:"decoder_type"`
	Package                string    `json:"package" bson:"package"`
	IucNumber              string    `json:"iuc_number" bson:"iuc_number"`
	Phone                  string    `json:"phone_number" bson:"phone_number"`
	Email                  string    `json:"email" bson:"email"`
	FullName               string    `json:"full_name" bson:"full_name"`
	Amount                 int       `json:"amount" bson:"amount"`
	TransactionProduct     string    `json:"transaction_product" bson:"transactionproduct"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	Token                  *string   `json:"token,omitempty" bson:"token,omitempty"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	RequestID              string    `json:"request_id" bson:"request_id"`
	ReferenceNumber        string    `json:"reference_number" bson:"reference_number"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
}
