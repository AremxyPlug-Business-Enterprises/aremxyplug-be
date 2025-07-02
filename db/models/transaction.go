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
