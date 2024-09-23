package dto

type Claims struct {
	PersonId string `json:"person_id"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenInput struct {
	Token string `json:"token"`
}

type UserResponse struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
}

type PasswordResetInput struct {
	Email string `json:"email"`
}
