package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/lenchik-en/lbs_server/internal/api/middleware"
	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
	"github.com/lenchik-en/lbs_server/internal/service"
)

// --- Shared fakes ---

type fakeAPIKeyRepo struct {
	keys   map[string]*model.APIKey
	getErr error
}

func (r *fakeAPIKeyRepo) GetByKey(_ context.Context, key string) (*model.APIKey, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.keys[key], nil
}

func (r *fakeAPIKeyRepo) Create(_ context.Context, scope model.KeyScope) (*model.APIKey, error) {
	return &model.APIKey{Key: "new", Scope: scope, IsActive: true}, nil
}

func (r *fakeAPIKeyRepo) Deactivate(_ context.Context, _ string) error { return nil }
func (r *fakeAPIKeyRepo) ListAll(_ context.Context) ([]*model.APIKey, error) {
	return nil, nil
}

type fakeSettingsRepo struct{}

func (r *fakeSettingsRepo) Load(_ context.Context) (*model.Settings, error) { return nil, nil }
func (r *fakeSettingsRepo) Save(_ context.Context, _ model.Settings) error  { return nil }

func newAPIKeySvc(keys map[string]*model.APIKey) *service.APIKeyService {
	return service.NewAPIKeyService(&fakeAPIKeyRepo{keys: keys})
}

func newAPIKeySvcWithErr() *service.APIKeyService {
	return service.NewAPIKeyService(&fakeAPIKeyRepo{getErr: errors.New("db error")})
}

func newAdminSvc(t *testing.T, login, pass string) *service.AdminService {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return service.NewAdminService(login, string(hash), model.Settings{}, &fakeSettingsRepo{})
}

func loginAdmin(t *testing.T, svc *service.AdminService, login, pass string) string {
	t.Helper()
	token, err := svc.Login(login, pass)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return token
}

// okHandler returns a 200 handler and a pointer to a "called" flag.
func okHandler() (http.Handler, *bool) {
	called := false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}), &called
}

// ==================== ParseBodyUUID ====================

func TestParseBodyUUID_ExtractsUUID(t *testing.T) {
	body := `{"sessionUUID":"abc-123-def"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var gotUUID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = middleware.SessionUUIDFromContext(r.Context())
		w.WriteHeader(200)
	})

	middleware.ParseBodyUUID(next).ServeHTTP(httptest.NewRecorder(), req)
	if gotUUID != "abc-123-def" {
		t.Errorf("expected UUID=abc-123-def, got %q", gotUUID)
	}
}

func TestParseBodyUUID_NoUUIDField(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"foo":"bar"}`))

	var gotUUID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = middleware.SessionUUIDFromContext(r.Context())
		w.WriteHeader(200)
	})

	middleware.ParseBodyUUID(next).ServeHTTP(httptest.NewRecorder(), req)
	if gotUUID != "" {
		t.Errorf("expected empty UUID, got %q", gotUUID)
	}
}

func TestParseBodyUUID_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`not json`))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Context has no UUID but body still readable.
		if middleware.SessionUUIDFromContext(r.Context()) != "" {
			t.Error("expected no UUID for invalid JSON")
		}
		w.WriteHeader(200)
	})

	w := httptest.NewRecorder()
	middleware.ParseBodyUUID(next).ServeHTTP(w, req)
	if !called {
		t.Error("handler must be called even for invalid JSON")
	}
}

func TestParseBodyUUID_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	w := httptest.NewRecorder()
	middleware.ParseBodyUUID(next).ServeHTTP(w, req)
	if !called {
		t.Error("handler must be called for empty body")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestParseBodyUUID_BodyRestoredForHandler(t *testing.T) {
	body := `{"sessionUUID":"x","data":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var handlerBody string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, len(body))
		n, _ := r.Body.Read(b)
		handlerBody = string(b[:n])
		w.WriteHeader(200)
	})

	middleware.ParseBodyUUID(next).ServeHTTP(httptest.NewRecorder(), req)
	if handlerBody != body {
		t.Errorf("body was not restored: got %q, want %q", handlerBody, body)
	}
}

// ==================== RequireAPIKey ====================

func TestRequireAPIKey_NoKeyNotRequired_PassesThrough(t *testing.T) {
	next, called := okHandler()
	h := middleware.RequireAPIKey(newAPIKeySvc(nil), model.ScopeLocate, func() bool { return false })(next)

	req := httptest.NewRequest(http.MethodPost, "/locate", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !*called || w.Code != http.StatusOK {
		t.Errorf("expected pass-through: called=%v code=%d", *called, w.Code)
	}
}

func TestRequireAPIKey_NoKeyRequired_Returns401(t *testing.T) {
	next, called := okHandler()
	h := middleware.RequireAPIKey(newAPIKeySvc(nil), model.ScopeLocate, func() bool { return true })(next)

	req := httptest.NewRequest(http.MethodPost, "/locate", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called || w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401: called=%v code=%d", *called, w.Code)
	}
}

func TestRequireAPIKey_InvalidKey_Returns403(t *testing.T) {
	next, called := okHandler()
	h := middleware.RequireAPIKey(
		newAPIKeySvc(map[string]*model.APIKey{}), // empty map → key not found
		model.ScopeLocate,
		func() bool { return true },
	)(next)

	req := httptest.NewRequest(http.MethodPost, "/locate", nil)
	req.Header.Set("X-Api-Key", "bad-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called || w.Code != http.StatusForbidden {
		t.Errorf("expected 403: called=%v code=%d", *called, w.Code)
	}
}

func TestRequireAPIKey_ValidKeyCorrectScope_PassesThrough(t *testing.T) {
	next, called := okHandler()
	keys := map[string]*model.APIKey{
		"good-key": {Key: "good-key", Scope: model.ScopeLocate, IsActive: true},
	}
	h := middleware.RequireAPIKey(
		newAPIKeySvc(keys), model.ScopeLocate, func() bool { return true },
	)(next)

	req := httptest.NewRequest(http.MethodPost, "/locate", nil)
	req.Header.Set("X-Api-Key", "good-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !*called || w.Code != http.StatusOK {
		t.Errorf("expected pass-through: called=%v code=%d", *called, w.Code)
	}
}

func TestRequireAPIKey_ValidKeyWrongScope_Returns403(t *testing.T) {
	next, called := okHandler()
	keys := map[string]*model.APIKey{
		"locate-key": {Key: "locate-key", Scope: model.ScopeLocate, IsActive: true},
	}
	h := middleware.RequireAPIKey(
		newAPIKeySvc(keys), model.ScopeUpdate, func() bool { return true },
	)(next)

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	req.Header.Set("X-Api-Key", "locate-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called || w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong scope: called=%v code=%d", *called, w.Code)
	}
}

func TestRequireAPIKey_BothScopeKey_PassesAnyScope(t *testing.T) {
	keys := map[string]*model.APIKey{
		"both-key": {Key: "both-key", Scope: model.ScopeBoth, IsActive: true},
	}
	for _, scope := range []model.KeyScope{model.ScopeLocate, model.ScopeUpdate} {
		next, called := okHandler()
		h := middleware.RequireAPIKey(
			newAPIKeySvc(keys), scope, func() bool { return true },
		)(next)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-Api-Key", "both-key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if !*called || w.Code != http.StatusOK {
			t.Errorf("both-key should pass scope=%s: called=%v code=%d", scope, *called, w.Code)
		}
	}
}

func TestRequireAPIKey_InactiveKey_Returns403(t *testing.T) {
	keys := map[string]*model.APIKey{
		"dead-key": {Key: "dead-key", Scope: model.ScopeLocate, IsActive: false},
	}
	next, _ := okHandler()
	h := middleware.RequireAPIKey(
		newAPIKeySvc(keys), model.ScopeLocate, func() bool { return true },
	)(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Api-Key", "dead-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for inactive key, got %d", w.Code)
	}
}

func TestRequireAPIKey_RepoError_Returns500(t *testing.T) {
	next, _ := okHandler()
	h := middleware.RequireAPIKey(
		newAPIKeySvcWithErr(), model.ScopeLocate, func() bool { return true },
	)(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Api-Key", "any-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for repo error, got %d", w.Code)
	}
}

// ==================== AdminAuth ====================

func TestAdminAuth_ValidCookie_PassesThrough(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	token := loginAdmin(t, svc, "admin", "pass")

	next, called := okHandler()
	h := middleware.AdminAuth(svc)(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AdminSessionCookie, Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !*called || w.Code != http.StatusOK {
		t.Errorf("expected pass-through: called=%v code=%d", *called, w.Code)
	}
}

func TestAdminAuth_NoCookie_RedirectsToLogin(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	next, called := okHandler()
	h := middleware.AdminAuth(svc)(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called {
		t.Error("handler must not be called without cookie")
	}
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %q", loc)
	}
}

func TestAdminAuth_InvalidCookie_RedirectsToLogin(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	next, called := okHandler()
	h := middleware.AdminAuth(svc)(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AdminSessionCookie, Value: "invalid-token"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called {
		t.Error("handler must not be called with invalid cookie")
	}
	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
}

func TestAdminAuth_AfterLogout_RedirectsToLogin(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	token := loginAdmin(t, svc, "admin", "pass")
	svc.Logout(token)

	next, called := okHandler()
	h := middleware.AdminAuth(svc)(next)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AdminSessionCookie, Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called {
		t.Error("handler must not be called after logout")
	}
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 after logout, got %d", w.Code)
	}
}

// ==================== AdminAPIAuth ====================

func TestAdminAPIAuth_ValidCookie_PassesThrough(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	token := loginAdmin(t, svc, "admin", "pass")

	next, called := okHandler()
	h := middleware.AdminAPIAuth(svc)(next)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AdminSessionCookie, Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !*called || w.Code != http.StatusOK {
		t.Errorf("expected pass-through: called=%v code=%d", *called, w.Code)
	}
}

func TestAdminAPIAuth_NoCookie_Returns401(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	next, called := okHandler()
	h := middleware.AdminAPIAuth(svc)(next)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called || w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401: called=%v code=%d", *called, w.Code)
	}
}

func TestAdminAPIAuth_InvalidCookie_Returns401(t *testing.T) {
	svc := newAdminSvc(t, "admin", "pass")
	next, called := okHandler()
	h := middleware.AdminAPIAuth(svc)(next)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AdminSessionCookie, Value: "bogus"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if *called || w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token: called=%v code=%d", *called, w.Code)
	}
}

// ==================== RateLimit ====================

func newLimiter(t *testing.T) *ratelimit.RateLimiter {
	l := ratelimit.New()
	t.Cleanup(l.Stop)
	return l
}

func TestRateLimit_ZeroPeriod_AlwaysPasses(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig { return middleware.RateLimitConfig{KeyPeriodMs: 0, AnonPeriodMs: 0} }

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", "same-key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("iteration %d: expected 200 for zero period, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_WithKey_FirstRequestPasses(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 10000, AnonPeriodMs: 0}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Api-Key", "key-alpha")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for first request, got %d", w.Code)
	}
}

func TestRateLimit_WithKey_SecondRequestBlocked(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 10000, AnonPeriodMs: 0}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	makeReq := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", "key-beta")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	if c := makeReq(); c != http.StatusOK {
		t.Errorf("first request: expected 200, got %d", c)
	}
	if c := makeReq(); c != http.StatusTooManyRequests {
		t.Errorf("second request: expected 429, got %d", c)
	}
}

func TestRateLimit_DifferentKeys_Independent(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 10000, AnonPeriodMs: 0}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	for _, key := range []string{"key1", "key2", "key3"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("first request for key=%q: expected 200, got %d", key, w.Code)
		}
	}
}

func TestRateLimit_AnonymousByIP_SecondRequestBlocked(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 0, AnonPeriodMs: 10000}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	makeReq := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	if c := makeReq(); c != http.StatusOK {
		t.Errorf("first anon request: expected 200, got %d", c)
	}
	if c := makeReq(); c != http.StatusTooManyRequests {
		t.Errorf("second anon request: expected 429, got %d", c)
	}
}

func TestRateLimit_XRealIP_UsedForAnon(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 0, AnonPeriodMs: 10000}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	makeReqWithRealIP := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", ip)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	if c := makeReqWithRealIP("5.5.5.5"); c != http.StatusOK {
		t.Errorf("first: expected 200, got %d", c)
	}
	if c := makeReqWithRealIP("5.5.5.5"); c != http.StatusTooManyRequests {
		t.Errorf("second with same X-Real-IP: expected 429, got %d", c)
	}
	// Different IP is not rate-limited.
	if c := makeReqWithRealIP("6.6.6.6"); c != http.StatusOK {
		t.Errorf("different X-Real-IP: expected 200, got %d", c)
	}
}

func TestRateLimit_XForwardedFor_UsesFirstIP(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 0, AnonPeriodMs: 10000}
	}

	next, _ := okHandler()
	h := middleware.RateLimit(limiter, cfg, "test")(next)

	// First request from 7.7.7.7 (via proxy chain).
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Forwarded-For", "7.7.7.7, 8.8.8.8, 9.9.9.9")
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("first XFF: expected 200, got %d", w1.Code)
	}

	// Second request with same first IP in X-Forwarded-For → blocked.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "7.7.7.7, 1.1.1.1") // same first IP
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second XFF same first IP: expected 429, got %d", w2.Code)
	}
}

func TestRateLimit_WithSessionUUID_UsedForAnon(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 0, AnonPeriodMs: 10000}
	}

	next, _ := okHandler()
	// Chain ParseBodyUUID before RateLimit so session UUID is injected into context.
	h := middleware.ParseBodyUUID(middleware.RateLimit(limiter, cfg, "test")(next))

	makeReqWithUUID := func(uuid string) int {
		body := `{"sessionUUID":"` + uuid + `"}`
		req := httptest.NewRequest(http.MethodPost, "/locate", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	if c := makeReqWithUUID("uuid-xyz"); c != http.StatusOK {
		t.Errorf("first UUID request: expected 200, got %d", c)
	}
	if c := makeReqWithUUID("uuid-xyz"); c != http.StatusTooManyRequests {
		t.Errorf("second same UUID request: expected 429, got %d", c)
	}
	// Different UUID passes.
	if c := makeReqWithUUID("uuid-other"); c != http.StatusOK {
		t.Errorf("different UUID: expected 200, got %d", c)
	}
}

func TestRateLimit_PrefixIsolatesEndpoints(t *testing.T) {
	limiter := newLimiter(t)
	cfg := func() middleware.RateLimitConfig {
		return middleware.RateLimitConfig{KeyPeriodMs: 10000, AnonPeriodMs: 0}
	}

	locateH := middleware.RateLimit(limiter, cfg, "locate")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	updateH := middleware.RateLimit(limiter, cfg, "update")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))

	makeReq := func(h http.Handler) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", "shared-key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	// First request to /locate uses the "locate" prefix.
	if c := makeReq(locateH); c != http.StatusOK {
		t.Errorf("locate first: expected 200, got %d", c)
	}
	// First request to /update with same key, different prefix → should pass.
	if c := makeReq(updateH); c != http.StatusOK {
		t.Errorf("update first (different prefix): expected 200, got %d", c)
	}
	// Second request to /locate → blocked.
	if c := makeReq(locateH); c != http.StatusTooManyRequests {
		t.Errorf("locate second: expected 429, got %d", c)
	}
}
