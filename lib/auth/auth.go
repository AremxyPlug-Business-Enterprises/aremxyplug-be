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

	privateKey, err := key_generator.GeneratePrivateKey(secret.JWTPrivateKey)
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
		// 1. Try validating the existing auth token
		if token := r.Header.Get("Authorization"); token != "" {
			if _, err := a.jwt.ValidateToken(token); err == nil {
				// Auth token still valid → proceed
				next.ServeHTTP(w, r)
				return
			}
		}

		// 2. Auth failed. Try to refresh via cookie.
		cookie, err := r.Cookie("refresh_token")
		if err == nil {
			if claims, err := a.jwt.ValidateToken(cookie.Value); err == nil {
				// Refresh token valid → issue new tokens
				newClaims := dto.Claims{PersonId: claims.ID}

				// Generate new auth token
				newAuthToken, err := a.jwt.GenerateToken(newClaims)
				if err == nil {
					w.Header().Set("x-new-auth-token", newAuthToken)
				}

				// Generate new refresh token
				if newRefresh, err := a.jwt.GenerateTokenWithExpiration(newClaims, tokengenerator.RefreshTokenDuration); err == nil {
					http.SetCookie(w, &http.Cookie{
						Name:     "refresh_token",
						Value:    newRefresh,
						MaxAge:   int(tokengenerator.RefreshTokenDuration.Seconds()),
						HttpOnly: true,
						Secure:   false,
						Path:     "/",
						SameSite: http.SameSiteNoneMode,
					})
				}

				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}
