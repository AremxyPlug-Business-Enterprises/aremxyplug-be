package models

type EduInfo struct {
	Exam_Type    string `json:"exam_type"`
	Phone_Number string `json:"phone_no"`
	Amount       string `json:"amount"`
	Email        string `json:"email"`
	Quantity     int    `json:"quantity"`
	Wallet_Type  string `json:"wallet_type"`
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
	OrderID         int      `json:"order_id" bson:"order_id"`
	Email           string   `json:"email" bson:"email"`
	Phone           string   `json:"phone_no" bson:"phone_no"`
	TransactionID   string   `json:"transaction_id"`
	Name            string   `json:"name" bson:"name"`
	ReferenceNumber string   `json:"reference_no" bson:"reference_no"`
	Product         string   `json:"product" bson:"product"`
	Amount          float64  `json:"amount" bson:"amount"`
	Exam_Type       string   `json:"exam_type" bson:"exam_type"`
	Description     string   `json:"description" bson:"description"`
	Status          string   `json:"status" bson:"status"`
	Pin_Generated   []string `json:"pins_generated" bson:"pins_generated"`
	CreatedAt       string   `json:"created_at" bson:"created_at"`
}
