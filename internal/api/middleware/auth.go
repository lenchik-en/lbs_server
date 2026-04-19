package middleware

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/service"
)

// RequireAPIKey проверяет X-Api-Key если require=true.
// При require=false пропускает запросы без ключа, но валидирует если ключ передан.
func RequireAPIKey(svc *service.APIKeyService, scope model.KeyScope, require bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Api-Key")
			if key == "" {
				if require {
					http.Error(w, "X-Api-Key required", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			ok, err := svc.Validate(r.Context(), key, scope)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "invalid or inactive API key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
