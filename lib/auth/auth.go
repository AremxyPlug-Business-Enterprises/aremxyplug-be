package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/redis"
	"github.com/aremxyplug-be/lib/key_generator"
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
)

type AuthConn struct {
	jwt         tokengenerator.TokenGenerator
	redisClient *redis.RedisConn
}

func NewAuthConn(secret *config.Secrets, client *redis.RedisConn) *AuthConn {
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
		redisClient: client,
		jwt: tokengenerator.New(
			publicKey,
			privateKey,
		),
	}
}

func (auth *AuthConn) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.jwt.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// update idle timeout
		sessionKey := fmt.Sprintf("session:%s", claims.ID)
		auth.redisClient.Client().Expire(context.Background(), sessionKey, 20*time.Minute)

		next.ServeHTTP(w, r)
	})
}
