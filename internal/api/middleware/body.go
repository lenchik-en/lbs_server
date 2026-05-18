package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type contextKey int

const sessionUUIDKey contextKey = iota

const maxBodyBytes = 1 << 20 // 1 MB

// ParseBodyUUID читает тело запроса, извлекает sessionUUID и кладёт в context.
// Тело восстанавливается, чтобы handler мог его прочитать повторно.
func ParseBodyUUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		var parsed struct {
			SessionUUID string `json:"sessionUUID"`
		}
		if json.Unmarshal(body, &parsed) == nil && parsed.SessionUUID != "" {
			r = r.WithContext(context.WithValue(r.Context(), sessionUUIDKey, parsed.SessionUUID))
		}

		next.ServeHTTP(w, r)
	})
}

// SessionUUIDFromContext возвращает sessionUUID, извлечённый ParseBodyUUID, или "".
func SessionUUIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionUUIDKey).(string)
	return v
}
