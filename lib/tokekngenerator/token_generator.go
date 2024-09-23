package tokengenerator

import (
	"crypto/rsa"
	"errors"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/types/dto"
	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidSigningMethod = errors.New("invalid token signing method")
	ErrInvalidToken         = errors.New("invalid token")
)

const (
	AuthTokenDuration    = 15 * time.Minute
	RefreshTokenDuration = 30 * time.Minute
)

type TokenGenerator interface {
	GenerateToken(data dto.Claims) (string, error)
	GenerateTokenWithExpiration(data dto.Claims, duration time.Duration) (string, error)
	ValidateToken(tokenString string) (*models.JWTClaims, error)
}

type jwtTokenGenerator struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func New(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) TokenGenerator {
	return &jwtTokenGenerator{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (j *jwtTokenGenerator) GenerateToken(data dto.Claims) (string, error) {
	return j.GenerateTokenWithExpiration(data, AuthTokenDuration)
}

func (j *jwtTokenGenerator) GenerateTokenWithExpiration(data dto.Claims, duration time.Duration) (string, error) {
	expirationTime := time.Now().Add(duration)
	claims := &models.JWTClaims{
		ID: data.PersonId,
		RegisteredClaims: &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(j.privateKey)
	return tokenString, err
}

func (j *jwtTokenGenerator) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	claims := &models.JWTClaims{}

	keyFunc := func(token *jwt.Token) (i interface{}, e error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return j.publicKey, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return &models.JWTClaims{
		ID: claims.ID,
	}, nil
}
