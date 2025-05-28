package telcom

import "time"

type DataInfo struct {
	UserID        string
	Network       int    `json:"network"`
	Plan          int    `json:"plan"`
	Mobile_Num    string `json:"mobile_number"`
	Name          string `json:"name"`
	Ported_number bool   `json:"Ported_number"`
	FullName      string
}

type DataResult struct {
	UserID          string    `json:"user_id" bson:"user_id"`
	OrderID         int       `json:"order_id" bson:"order_id"`
	TransactionID   string    `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber string    `json:"reference_number" bson:"reference_number"`
	Network         string    `json:"network" bson:"network"`
	FullName        string    `json:"full_name" bson:"full_name"`
	PlanName        string    `json:"plan_name" bson:"plan_name"`
	Plan_Amount     string    `json:"plan_amount" bson:"plan_amount"`
	Status          string    `json:"Status" bson:"status"`
	Name            string    `json:"Name" bson:"name"`
	Phone_Number    string    `json:"Phone_Number" bson:"phone_number"`
	CreatedAt       time.Time `json:"CreatedAt" bson:"created_at"`
	ApiID           int       `bson:"apiID"`
}

type APIResponse struct {
	Id int `json:"id"`
	//Network       string `json:"network" bson:"network"`
	Plan_Name     string `json:"plan_name"`
	Plan_network  string `json:"plan_network"`
	Plan_amount   string `json:"plan_amount"`
	Mobile_number string `json:"mobile_number"`
	Ident         string `json:"ident"`
	Status        string `json:"Status"`
}
