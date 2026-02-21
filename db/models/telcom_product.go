package models

import "database/sql"

type Product struct {
	Product_ID int
	Network_ID int
	Plan_Type  string
}

type Plan struct {
	ID           int
	PlanID       sql.NullInt64
	ProductID    int
	Size         string
	Amount       float64
	Validity     string
	ProviderID   int
	PlanType     string
	ProfitMargin float64
}

type PlanUpdate struct {
	Amount   float64 `json:"amount,omitempty"`
	Validity string  `json:"validity,omitempty"`
	Size     string  `json:"size,omitempty"`
}

type AirtimeProduct struct {
	Network           string
	Provider_Discount float64
	Customer_Discount float64
}
