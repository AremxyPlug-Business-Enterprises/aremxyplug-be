package key_generator

import (
	"crypto/rsa"

	"github.com/dgrijalva/jwt-go"
)

func GeneratePublicKey(publicKey string) (*rsa.PublicKey, error) {
	tokenGeneratorPublicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
	if err != nil {
		return nil, err
	}
	return tokenGeneratorPublicKey, nil
}

func GeneratePrivateKey(privateKey string) (*rsa.PrivateKey, error) {
	tokenGeneratorPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	return tokenGeneratorPrivateKey, nil
}
