package models

import "github.com/golang-jwt/jwt/v4"

// JWTClaims defines the custom REMITING claims for JWT tokens.
type JWTClaims struct {
	*jwt.RegisteredClaims
	ID    string `json:"id"`
	Email string `json:"email"`
}
