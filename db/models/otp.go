package models

import "time"

type OTP struct {
	Secret   string    `bson:"secret"`
	Channel  string    `bson:"channel"`
	Target   string    `bson:"target"`
	ExpireAt time.Time `bson:"expireAt"`
}

type SMSOTP struct {
	PinID    string    `bson:"pinID"`
	Phone    string    `bson:"phone"`
	ExpireAt time.Time `bson:"expireAt"`
}
