package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/api/middleware"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

// errReader возвращает ошибку при первом чтении — для теста ParseBodyUUID.
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, errors.New("forced read error") }

// TestParseBodyUUID_ReadError_Returns400 покрывает ветку io.ReadAll error.
func TestParseBodyUUID_ReadError_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", errReader{})
	next, called := okHandler()
	w := httptest.NewRecorder()

	middleware.ParseBodyUUID(next).ServeHTTP(w, req)

	if *called {
		t.Error("handler must not be called when body read fails")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on read error, got %d", w.Code)
	}
}

// TestRateLimit_XForwardedFor_SingleIP покрывает ветку без запятой в XFF.
func TestRateLimit_XForwardedFor_SingleIP(t *testing.T) {
	limiter := ratelimit.New()
	t.Cleanup(limiter.Stop)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{AnonPeriodMs: 10000}
	}
	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "xff_single")(next)

	// Первый запрос — должен пройти.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Forwarded-For", "4.4.4.4") // одиночный IP, без запятой
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("single XFF first: expected 200, got %d", w1.Code)
	}

	// Второй запрос с тем же IP — должен быть заблокирован.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "4.4.4.4")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("single XFF second: expected 429, got %d", w2.Code)
	}
}

// TestRateLimit_MalformedRemoteAddr покрывает ветку SplitHostPort error в realIP.
func TestRateLimit_MalformedRemoteAddr(t *testing.T) {
	limiter := ratelimit.New()
	t.Cleanup(limiter.Stop)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{AnonPeriodMs: 10000}
	}
	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "bad_remote")(next)

	// RemoteAddr без порта — net.SplitHostPort вернёт ошибку,
	// realIP должна вернуть RemoteAddr как есть.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4" // нет порта
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("malformed RemoteAddr first: expected 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("malformed RemoteAddr second: expected 429, got %d", w2.Code)
	}
}
