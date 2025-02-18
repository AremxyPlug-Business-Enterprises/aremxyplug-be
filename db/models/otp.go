package models

import "time"

type OTP struct {
	Secret   string    `bson:"secret"`
	Email    string    `bson:"email"`
	ExpireAt time.Time `bson:"expireAt"`
}

type SMSOTP struct {
	PinID    string    `bson:"pinID"`
	Phone    string    `bson:"phone"`
	ExpireAt time.Time `bson:"expireAt"`
}
