package models

import "time"

type TransactionResponse struct {
	Transactions []TransactionItem `json:"transactions"`
	TotalCount   int               `json:"total_count"`
	TotalInflow  float64           `json:"total_inflow"`
	TotalOutflow float64           `json:"total_outflow"`
}

type TransactionItem struct {
	Product     string    `bson:"product" json:"product"`
	Description string    `bson:"description" json:"description"`
	OrderID     int       `bson:"order_id" json:"order_id"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	Status      string    `bson:"status" json:"status"`
	FlowType    string    `bson:"flowType" json:"flow_type"`
	Amount      float64   `bson:"amountDecimal" json:"amount"`
}

type TotalsAggregation struct {
	TotalCount   int     `bson:"totalCount"`
	TotalInflow  float64 `bson:"totalInflow"`
	TotalOutflow float64 `bson:"totalOutflow"`
}

type SalesSummary struct {
	Summary      []SalesSummaryItem `json:"summary"`
	TotalCount   int                `json:"total"`
	TotalInflow  float64            `json:"total_inflow"`
	TotalOutflow float64            `json:"total_outflow"`
}

type SalesSummaryItem struct {
	Product     string    `bson:"product" json:"product"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	Quantity    int       `bson:"quantity" json:"quantity"`
	TotalAmount float64   `bson:"totalAmount" json:"total_amount"`
}
