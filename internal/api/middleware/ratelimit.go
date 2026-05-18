package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

// RateLimitConfig задаёт периоды для одного эндпоинта.
// KeyPeriodMs  — минимальный период (мс) между запросами одного API-ключа.
// AnonPeriodMs — минимальный период (мс) между запросами одного sessionUUID или IP (без ключа).
// 0 = отключено.
type RateLimitConfig struct {
	KeyPeriodMs  int
	AnonPeriodMs int
}

// realIP возвращает реальный IP клиента: X-Real-IP → первый X-Forwarded-For → RemoteAddr.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx != -1 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimit реализует логику "с ключом OR без ключа":
//   - есть X-Api-Key → ограничение по значению ключа
//   - нет X-Api-Key, есть sessionUUID → ограничение по UUID
//   - нет ни того ни другого → ограничение по IP (fallback)
//
// cfgFn вызывается на каждый запрос — позволяет менять лимиты через UI без перезапуска.
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
				identifier := SessionUUIDFromContext(r.Context())
				if identifier == "" {
					identifier = realIP(r)
				}
				if !limiter.Allow(prefix+":anon:"+identifier, cfg.AnonPeriodMs) {
					http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
