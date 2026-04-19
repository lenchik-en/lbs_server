package api

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/api/handler"
	"github.com/lenchik-en/lbs_server/internal/api/middleware"
	"github.com/lenchik-en/lbs_server/internal/config"
	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
	"github.com/lenchik-en/lbs_server/internal/service"
)

// NewMux собирает HTTP-маршрутизатор из сервисов.
func NewMux(
	cfg *config.Config,
	limiter *ratelimit.RateLimiter,
	locateSvc *service.LocateService,
	updateSvc *service.UpdateService,
	refreshSvc *service.RefreshService,
	apiKeySvc *service.APIKeyService,
	demo *handler.DemoHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Healthcheck
	mux.HandleFunc("/healthz", handler.Health)

	// /locate
	locateRL := middleware.RateLimit(limiter,
		middleware.RateLimitConfig{
			KeyPeriodMs:     cfg.LocateMinPeriodKey,
			UUIDPeriodMs:    cfg.LocateMinPeriodUUID,
			KeyUUIDPeriodMs: cfg.LocateMinPeriodKeyUUID,
		},
		"locate",
		func(r *http.Request) string { return r.Header.Get("X-Api-Key") },
		func(r *http.Request) string { return "" }, // UUID из body — читается в хендлере
	)
	locateAuth := middleware.RequireAPIKey(apiKeySvc, model.ScopeLocate, cfg.RequireKeyLocate)
	mux.Handle("/locate", locateAuth(locateRL(handler.NewLocateHandler(locateSvc))))

	// /update
	updateRL := middleware.RateLimit(limiter,
		middleware.RateLimitConfig{
			KeyPeriodMs:     cfg.UpdateMinPeriodKey,
			UUIDPeriodMs:    cfg.UpdateMinPeriodUUID,
			KeyUUIDPeriodMs: cfg.UpdateMinPeriodKeyUUID,
		},
		"update",
		func(r *http.Request) string { return r.Header.Get("X-Api-Key") },
		func(r *http.Request) string { return "" },
	)
	updateAuth := middleware.RequireAPIKey(apiKeySvc, model.ScopeUpdate, cfg.RequireKeyUpdate)
	mux.Handle("/update", updateAuth(updateRL(handler.NewUpdateHandler(updateSvc))))

	// /refresh
	mux.Handle("/refresh", handler.NewRefreshHandler(refreshSvc))

	// /demo/cells
	mux.Handle("/demo/cells", demo)

	// /admin/keys
	keyHandler := handler.NewAPIKeyHandler(apiKeySvc)
	mux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			keyHandler.List(w, r)
		case http.MethodPost:
			keyHandler.Create(w, r)
		case http.MethodDelete:
			keyHandler.Revoke(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

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
