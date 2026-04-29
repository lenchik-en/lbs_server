package middleware

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

// RateLimitConfig задаёт периоды для одного эндпоинта.
// KeyPeriodMs  — минимальный период (мс) между запросами одного API-ключа.
// AnonPeriodMs — минимальный период (мс) между запросами одного sessionUUID (без ключа).
// 0 = отключено.
type RateLimitConfig struct {
	KeyPeriodMs  int
	AnonPeriodMs int
}

// RateLimit реализует логику "с ключом OR без ключа":
//   - есть X-Api-Key → ограничение по значению ключа
//   - нет X-Api-Key  → ограничение по sessionUUID из контекста (ParseBodyUUID должен быть выше в цепочке)
//
// cfgFn вызывается на каждый запрос — позволяет менять лимиты через UI без перезапуска.
// todo: клиент без ключа и без uuid не проверяется (сделать проверку по айпт как fallback)
func RateLimit(
	limiter *ratelimit.RateLimiter,
	cfgFn func() RateLimitConfig,
	prefix string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := cfgFn()
			if key := r.Header.Get("X-Api-Key"); key != "" {
				if !limiter.Allow(prefix+":key:"+key, cfg.KeyPeriodMs) {
					http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			} else {
				uuid := SessionUUIDFromContext(r.Context())
				if uuid != "" && !limiter.Allow(prefix+":anon:"+uuid, cfg.AnonPeriodMs) {
					http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
