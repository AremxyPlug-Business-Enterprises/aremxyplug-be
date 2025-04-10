package auth

import (
	"log"
	"net/http"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/lib/key_generator"
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
	"github.com/aremxyplug-be/types/dto"
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

func (a *AuthConn) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the token from the header
		token := r.Header.Get("Authorization")
		// Validate the token
		_, err := a.jwt.ValidateToken(token)
		if err != nil {
			// Check for the refresh_token in cookies
			cookie, err := r.Cookie("refresh_token")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized: invalid or missing token and refresh token: " + err.Error()))
				return
			}

			// Validate the refresh token
			claims, err := a.jwt.ValidateToken(cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized: invalid refresh token: " + err.Error()))
				return
			}

			newClaims := dto.Claims{
				PersonId: claims.ID,
			}

			// Generate a new auth token
			newAuthToken, err := a.jwt.GenerateToken(newClaims)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error generating new auth token: " + err.Error()))
				return
			}

			newRefreshToken, err := a.jwt.GenerateTokenWithExpiration(newClaims, tokengenerator.RefreshTokenDuration)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error generating new token: " + err.Error()))
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    newRefreshToken,
				MaxAge:   1800,
				HttpOnly: true,
				Secure:   false,
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			})

			// Set the new auth token in the response header
			w.Header().Set("x-new-auth-header", newAuthToken)
		}

		next.ServeHTTP(w, r)
	})
}
