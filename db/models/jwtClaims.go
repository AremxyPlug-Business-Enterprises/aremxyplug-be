package models

import "github.com/golang-jwt/jwt/v4"

// JWTClaims struct
type JWTClaims struct {
	*jwt.RegisteredClaims
	ID    string `json:"id"`
	Email string `json:"email"`
}
