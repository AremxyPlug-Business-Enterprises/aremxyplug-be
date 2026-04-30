package models

import "database/sql"

type Product struct {
	Product_ID int
	Network_ID int
	Plan_Type  string
}

type Plan struct {
	ID           int           `json:"ID,omitempty"`
	PlanID       sql.NullInt64 `json:"PlanID,omitempty"`
	ProductID    int           `json:"ProductID,omitempty"`
	Size         string        `json:"Size,omitempty"`
	Amount       float64       `json:"Amount,omitempty"`
	Validity     string        `json:"Validity,omitempty"`
	ProviderID   int           `json:"ProviderID,omitempty"`
	PlanType     string        `json:"PlanType,omitempty"`
	ProfitMargin float64       `json:"ProfitMargin,omitempty"`
}

type PlanUpdate struct {
	Amount   float64 `json:"amount,omitempty"`
	Validity string  `json:"validity,omitempty"`
	Size     string  `json:"size,omitempty"`
}

type AirtimeProduct struct {
	ID                int
	Network           string
	ProviderID        int
	Provider_Discount float64
	Customer_Discount float64
	Profit_Margin     float64
}
