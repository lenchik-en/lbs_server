package api

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/api/handler"
	"github.com/lenchik-en/lbs_server/internal/api/middleware"
	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
	"github.com/lenchik-en/lbs_server/internal/service"
)

// NewMux собирает HTTP-маршрутизатор из сервисов.
func NewMux(
	adminSvc *service.AdminService,
	limiter *ratelimit.RateLimiter,
	locateSvc *service.LocateService,
	updateSvc *service.UpdateService,
	refreshSvc *service.RefreshService,
	apiKeySvc *service.APIKeyService,
) http.Handler {
	mux := http.NewServeMux()

	// Healthcheck
	mux.HandleFunc("/healthz", handler.Health)

	// /locate — requireFn читает актуальные настройки из AdminService
	locateRL := middleware.RateLimit(limiter,
		func() middleware.RateLimitConfig {
			s := adminSvc.GetSettings()
			return middleware.RateLimitConfig{KeyPeriodMs: s.LocateMinPeriodKey, AnonPeriodMs: s.LocateMinPeriodAnon}
		},
		"locate",
	)
	locateAuth := middleware.RequireAPIKey(apiKeySvc, model.ScopeLocate,
		func() bool { return adminSvc.GetSettings().RequireKeyLocate },
	)
	// ParseBodyUUID → RateLimit → Auth → Handler
	mux.Handle("/locate", middleware.ParseBodyUUID(locateRL(locateAuth(handler.NewLocateHandler(locateSvc)))))

	// /update
	updateRL := middleware.RateLimit(limiter,
		func() middleware.RateLimitConfig {
			s := adminSvc.GetSettings()
			return middleware.RateLimitConfig{KeyPeriodMs: s.UpdateMinPeriodKey, AnonPeriodMs: s.UpdateMinPeriodAnon}
		},
		"update",
	)
	updateAuth := middleware.RequireAPIKey(apiKeySvc, model.ScopeUpdate,
		func() bool { return adminSvc.GetSettings().RequireKeyUpdate },
	)
	// ParseBodyUUID → RateLimit → Auth → Handler
	mux.Handle("/update", middleware.ParseBodyUUID(updateRL(updateAuth(handler.NewUpdateHandler(updateSvc)))))

	// /refresh (публичный — только для внутреннего использования)
	mux.Handle("/refresh", handler.NewRefreshHandler(refreshSvc))

	// --- Админ-панель ---
	adminH := handler.NewAdminHandler(adminSvc, apiKeySvc, refreshSvc)
	adminAuth := middleware.AdminAuth(adminSvc)

	mux.HandleFunc("/admin/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminH.LoginPage(w, r)
		case http.MethodPost:
			adminH.LoginSubmit(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/logout", adminH.Logout)

	// Все остальные /admin/* — под сессионной проверкой
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/", adminH.Dashboard)
	adminMux.HandleFunc("/admin/settings", adminH.SaveSettings)
	adminMux.HandleFunc("/admin/refresh", adminH.RunRefresh)
	adminMux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminH.KeysPage(w, r)
		case http.MethodPost:
			adminH.CreateKey(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/admin/keys/revoke", adminH.RevokeKey)

	mux.Handle("/admin/", adminAuth(adminMux))

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Api-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
