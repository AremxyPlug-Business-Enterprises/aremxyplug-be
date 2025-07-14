package models

import "database/sql"

type Product struct {
	Product_ID int
	Network_ID int
	Plan_Type  string
}

type Plan struct {
	ID         int           `json:"plan_id"`
	PlanID     sql.NullInt64 `json:"-"`           // external plan ID (nullable, may be missing initially)
	ProductID  int           `json:"product_id"`  // FK to products table
	Size       string        `json:"size"`        // e.g., "1GB", "500MB"
	Amount     float64       `json:"amount"`      // price
	Validity   string        `json:"validity"`    // e.g., "30 days"
	ProviderID int           `json:"provider_id"` // FK to api_providers  // updated timestamp
	PlanType   string        `json:"plan_type"`   // e.g., "data", "voice", "sms"
}

type PlanUpdate struct {
	Amount   float64 `json:"amount,omitempty"`
	Validity string  `json:"validity,omitempty"`
	Size     string  `json:"size,omitempty"`
}
