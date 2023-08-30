package models

type DataInfo struct {
	Network       int    `json:"network"`
	Network_id    int    `json:"newtork_id"`
	Plan          int    `json:"plan"`
	Plan_id       string `json:"plan_id"`
	Mobile_Num    string `json:"mobile_number"`
	Ported_number bool   `json:"Ported_number"`
	Name          string `json:"name"`
}

type DataResult struct {
	OrderID         int    `json:"id" bson:"order_id"`
	TransactionID   string `json:"transaction_id" bson:"transaction_id"`
	ReferenceNumber string `json:"reference_number" bson:"reference_number"`
	Network         string `json:"network" bson:"network"`
	Username        string `json:"username" bson:"username"`
	PlanName        string `json:"plan_name" bson:"plan_name"`
	Plan_Amount     string `json:"plan_amount" bson:"plan_amount"`
	Status          string `json:"Status" bson:"status"`
	Name            string `json:"Name" bson:"name"`
	Phone_Number    string `json:"Phone_Number" bson:"phone_number"`
	CreatedAt       string `json:"CreatedAt" bson:"created_at"`
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
