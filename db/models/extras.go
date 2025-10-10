package models

import "time"

type Referral struct {
	UserID     string    `json:"user_id" bson:"user_id"`
	ReferrerID string    `json:"referrerID" bson:"referrerID"`
	ReferredAt time.Time `json:"referred_at" bson:"referred_at"`
	IsVerified bool      `json:"is_verified" bson:"is_verified"`
	IsActive   bool      `json:"is_active" bson:"is_active"`
}

type ReferredUserInfo struct {
	UserID     string    `json:"referred_id" bson:"referred_id"`
	FullName   string    `json:"full_name" bson:"full_name"`
	Username   string    `json:"username" bson:"username"`
	IsActive   bool      `json:"is_active" bson:"is_active"`
	ReferredAt time.Time `json:"referred_at" bson:"referred_at"`
}

type Points struct {
	UserID    string    `json:"user_id" bson:"user_id"`
	Balance   int       `json:"balance" bson:"balance"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type PointTransaction struct {
	UserID              string    `json:"user_id" bson:"user_id"`
	TransactionType     string    `json:"transaction_type" bson:"transaction_type"`         // e.g., "earn", "redeem"
	PointEarned         int       `json:"point_earned" bson:"point_earned"`                 // Can be negative for redemption
	OriginalTransaction string    `json:"original_transaction" bson:"original_transaction"` // Reference to original txn, like txn ID
	Source              string    `json:"source" bson:"source"`                             // e.g., "referral", "airtime_purchase"
	TransactionID       string    `json:"transaction_id" bson:"transaction_id"`             // Unique ID for this point transaction
	CreatedAt           time.Time `json:"created_at" bson:"created_at"`
}

type PointSummary struct {
	EarnedPoints      int `json:"earned_points"`
	TransactionPoints int `json:"transaction_points"`
	ReferralPoints    int `json:"referral_points"`
	AvailablePoints   int `json:"available_points"`
}

type PointRedeem struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Points_Redeemed        string    `json:"points_redeemed" bson:"points_redeemed"`
	Amount_Redeemed        string    `json:"amount_redeemed" bson:"amount_redeemed"`
	Redeemed_Rate          string    `json:"redeemed_rate" bson:"redeemed_rate"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	OrderID                int       `json:"order_id" bson:"order_id"`
	TransactionID          string    `json:"transaction_id" bson:"transaction_id"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
}
