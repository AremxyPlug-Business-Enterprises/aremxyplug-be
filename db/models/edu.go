package models

import "time"

type EduInfo struct {
	UserID       string
	Exam_Type    string `json:"exam_type"`
	Phone_Number string `json:"phone_no"`
	Amount       string `json:"amount"`
	Email        string `json:"email"`
	Quantity     int    `json:"quantity"`
	Name         string
}

type EduApiResponse struct {
	Message          string  `json:"message"`
	Amount           float64 `json:"amount"`
	Date             string  `json:"transaction_date"`
	Status           string  `json:"status"`
	Reference        string  `json:"reference_no"`
	Pin1             string  `json:"pin"`
	Pin2             string  `json:"pin2,omitempty"`
	Pin3             string  `json:"pin3,omitempty"`
	Pin4             string  `json:"pin4,omitempty"`
	Pin5             string  `json:"pin5,omitempty"`
	Pin6             string  `json:"pin6,omitempty"`
	Pin7             string  `json:"pin7,omitempty"`
	Pin8             string  `json:"pin8,omitempty"`
	Pin9             string  `json:"pin9,omitempty"`
	Pin10            string  `json:"pin10,omitempty"`
	Success_Response string  `json:"success"`
}

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
}

type EduRecord struct {
	ID     int
	Amount string
	Name   string
}
