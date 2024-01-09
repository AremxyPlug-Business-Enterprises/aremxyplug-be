package auth

import (
	"log"
	"net/http"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/lib/key_generator"
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
)

type AuthConn struct {
	jwt tokengenerator.TokenGenerator
}

func NewAuthConn(secret *config.Secrets) *AuthConn {
	publicKey, err := key_generator.GeneratePublicKey(secret.JWTPublicKey)
	if err != nil {
		log.Println(err)
	}

	privateKey, err := key_generator.GeneratePrivateKey(secret.JWTPublicKey)
	if err != nil {
		// do something with the error
		log.Println(err)
	}
	return &AuthConn{
		jwt: tokengenerator.New(
			publicKey,
			privateKey,
		),
	}
}

// authorisation middleware
func (a *AuthConn) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//get the token from the header
		token := r.Header.Get("Authorization")
		//validate the token
		_, err := a.jwt.ValidateToken(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})

}
