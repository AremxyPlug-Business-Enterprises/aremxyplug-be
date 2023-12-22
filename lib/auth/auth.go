package auth

import (
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
	"net/http"
)

type AuthConn struct {
	jwt tokengenerator.TokenGenerator
}

func NewAuthConn(jwt tokengenerator.TokenGenerator) *AuthConn {
	return &AuthConn{
		jwt: jwt,
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
