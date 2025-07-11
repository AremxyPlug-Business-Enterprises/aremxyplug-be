package models

import "time"

type CollectionResult struct {
	TimeMap      map[string]TimeStat `json:"hourlyMap"`
	TotalAmount  float64             `json:"totalAmount"`
	TotalCount   int                 `json:"totalCount"`
	Transactions []TransactionPoint  `json:"transactions"`
}

type TimeStat struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type TransactionPoint struct {
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	Amount    float64   `bson:"amount" json:"amount"`
}

type StatsResponse struct {
	TotalInflowCount    int                `json:"totalInflowCount"`
	TotalInflowAmount   float64            `json:"totalInflowAmount"`
	TotalOutflowCount   int                `json:"totalOutflowCount"`
	TotalOutflowAmount  float64            `json:"totalOutflowAmount"`
	Inflow              []TimeStat         `json:"inflow"`
	Outflow             []TimeStat         `json:"outflow"`
	InflowTransactions  []TransactionPoint `json:"inflowTransactions"`
	OutflowTransactions []TransactionPoint `json:"outflowTransactions"`
}
