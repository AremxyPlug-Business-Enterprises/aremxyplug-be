package dto

type Claims struct {
	PersonId string `json:"person_id"`
	Email    string `json:"email"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenInput struct {
	Token string `json:"token"`
}

type UserResponse struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type PasswordResetInput struct {
	Email string `json:"email"`
}
