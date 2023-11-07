package models

// User is the model that governs all notes objects retrived or inserted into the DB
type User struct {
	ID             string `json:"id" bson:"id"`
	FullName       string `json:"fullname" bson:"fullname" validate:"required,min=2,max=100"`
	Email          string `json:"email" bson:"email" validate:"email,required"`
	Username       string `json:"username" bson:"username" validate:"required,min=2,max=100"`
	Password       string `json:"password" bson:"password" validate:"required,min=6"`
	PhoneNumber    string `json:"phone_number" bson:"phonenumber" validate:"required"`
	Country        string `json:"country" bson:"country" validate:"required"`
	InvitationCode string `json:"invitation_code" bson:"invitation_Code"`
	CreatedAt      int64  `json:"created_at" bson:"created_at"`
	UpdatedAt      int64  `json:"updated_at" bson:"updated_at"`
	IsVerified     bool   `json:"is_verified" bson:"is_verified"`
}
