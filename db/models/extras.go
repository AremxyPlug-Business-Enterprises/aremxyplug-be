package models

import "time"

type Referral struct {
	UserID     string    `json:"user_id" bson:"user_id"`
	ReferrerID string    `json:"referrerID" bson:"referrerID"`
	ReferredAt time.Time `json:"referred_at" bson:"referred_at"`
	IsActive   bool      `json:"is_active" bson:"is_active"`
}

type ReferredUserInfo struct {
	UserID     string    `json:"user_id" bson:"user_id"`
	FullName   string    `json:"full_name" bson:"full_name"`
	Email      string    `json:"email" bson:"email"`
	IsActive   bool      `json:"is_active" bson:"is_active"`
	ReferredAt time.Time `json:"referred_at" bson:"referred_at"`
}

type Points struct {
	UserID  string `json:"user_id" bson:"user_id"`
	Balance int    `json:"balance" bson:"balance"`
}

type HourlyStat struct {
	Hour   int     `json:"hour"`
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
	InflowHourly        []HourlyStat       `json:"inflowHourly"`
	OutflowHourly       []HourlyStat       `json:"outflowHourly"`
	InflowTransactions  []TransactionPoint `json:"inflowTransactions"`
	OutflowTransactions []TransactionPoint `json:"outflowTransactions"`
}
