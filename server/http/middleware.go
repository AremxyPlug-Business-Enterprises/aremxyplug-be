package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/redis"
)

type RateLimitConfig struct {
	Limiter       *redis.RedisConn
	MaxPerIP      int
	IPWindow      time.Duration
	MaxPerUser    int
	UserWindow    time.Duration
	PerRouteLimit map[string]PerRouteConfig
}

type PerRouteConfig struct {
	Limit  int
	Window time.Duration
}

func RateLimitMiddleware(cfg RateLimitConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			// ============================
			// 1. GET Client IP
			// ============================
			ip := parseRealIP(r)

			// ============================
			// 2. GLOBAL IP LIMITING
			// ============================
			ipKey := "rl:ip:" + ip
			allowed, count, _ := cfg.Limiter.Allow(ctx, ipKey, cfg.MaxPerIP, cfg.IPWindow)
			if !allowed {
				http.Error(w, "Too many requests from this IP", http.StatusTooManyRequests)
				return
			}
			_ = count

			// ============================
			// 3. PER-USER LIMITING
			// ============================
			userID := r.Context().Value("user_id") // set by your auth middleware
			if userID != nil {
				uKey := "rl:user:" + userID.(string)
				allowed, _, _ = cfg.Limiter.Allow(ctx, uKey, cfg.MaxPerUser, cfg.UserWindow)
				if !allowed {
					http.Error(w, "Too many requests for this user", http.StatusTooManyRequests)
					return
				}
			}

			// ============================
			// 4. PER-ROUTE LIMITING
			// ============================
			if routeCfg, ok := cfg.PerRouteLimit[r.URL.Path]; ok {
				rKey := "rl:route:" + r.URL.Path + ":" + ip
				allowed, _, _ = cfg.Limiter.Allow(ctx, rKey, routeCfg.Limit, routeCfg.Window)
				if !allowed {
					http.Error(w, "Rate limit exceeded for this endpoint", http.StatusTooManyRequests)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Get real IP behind proxy
func parseRealIP(r *http.Request) string {
	h := r.Header.Get("X-Forwarded-For")
	if h != "" {
		parts := strings.Split(h, ",")
		return strings.TrimSpace(parts[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
