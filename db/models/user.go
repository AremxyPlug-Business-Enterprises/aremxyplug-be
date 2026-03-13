package models

import "time"

// User is the model that governs all notes objects retrived or inserted into the DB
type User struct {
	ID              string    `json:"id" bson:"id"`
	FullName        string    `json:"fullname" bson:"fullname" validate:"required,min=2,max=100"`
	Email           string    `json:"email" bson:"email" validate:"email,required"`
	Username        string    `json:"username" bson:"username" validate:"required,min=2,max=100"`
	Password        string    `json:"password" bson:"password" validate:"required,min=6"`
	PhoneNumber     string    `json:"phone_number" bson:"phonenumber" validate:"required"`
	Country         string    `json:"country" bson:"country" validate:"required"`
	InvitationCode  string    `json:"invitation_code" bson:"invitation_Code"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
	BVN             string    `json:"bvn" bson:"bvn"`
	BVNPhone        string    `json:"bvn_phone,omitempty" bson:"bvn_phone,omitempty"`
	NIN             string    `json:"nin" bson:"nin"`
	DOB             time.Time `json:"birth_date" bson:"birth_date"`
	IsVerified      bool      `json:"is_verified" bson:"is_verified"`
	HasPin          bool      `json:"has_Pin" bson:"has_Pin"`
	HasBVN          bool      `json:"has_bvn" bson:"has_bvn"`
	HasNIN          bool      `json:"has_nin" bson:"has_nin"`
	HasVirtualNuban bool      `json:"has_virtual_nuban" bson:"has_virtual_nuban"`
	ExpireAt        time.Time `bson:"expireAt"`
	ReferralCount   int       `json:"referral_count" bson:"referral_count"`
	LastTransaction time.Time `json:"last_transaction" bson:"last_transaction"`
	Address         string    `json:"address" bson:"address"`
	PostalCode      string    `json:"postal_code" bson:"postal_code"`
	Gender          string    `json:"gender" bson:"gender"`
	Beta            bool      `json:"beta" bson:"beta"`
}
