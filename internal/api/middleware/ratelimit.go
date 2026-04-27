package middleware

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

// RateLimitConfig задаёт периоды для одного эндпоинта.
type RateLimitConfig struct {
	KeyPeriodMs     int
	UUIDPeriodMs    int
	KeyUUIDPeriodMs int
}

// RateLimit проверяет три уровня rate limiting: по API-ключу, UUID и паре (ключ, UUID).
// keyFn и uuidFn извлекают идентификаторы из запроса.
func RateLimit(
	limiter *ratelimit.RateLimiter,
	cfg RateLimitConfig,
	prefix string,
	keyFn func(*http.Request) string,
	uuidFn func(*http.Request) string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := keyFn(r)
			sessionUUID := uuidFn(r)

			if apiKey != "" && !limiter.Allow(prefix+":key:"+apiKey, cfg.KeyPeriodMs) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			if sessionUUID != "" && !limiter.Allow(prefix+":uuid:"+sessionUUID, cfg.UUIDPeriodMs) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			if apiKey != "" && sessionUUID != "" && !limiter.Allow(prefix+":key+uuid:"+apiKey+":"+sessionUUID, cfg.KeyUUIDPeriodMs) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
