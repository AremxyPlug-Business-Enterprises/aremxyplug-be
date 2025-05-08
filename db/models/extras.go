package models

import "time"

type Referral struct {
	UserID     string    `json:"user_id"`
	ReferrerID string    `json:"referrerID" bson:"referrerID"`
	ReferredAt time.Time `json:"referred_at" bson:"referred_at"`
	IsActive   bool      `json:"is_active" bson:"is_active"`
}

type ReferredUserInfo struct {
	UserID     string    `json:"user_id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	IsActive   bool      `json:"is_active"`
	ReferredAt time.Time `json:"referred_at"`
}

type Points struct {
	UserID  string `json:"user_id"`
	Balance int    `json:"balance"`
}
