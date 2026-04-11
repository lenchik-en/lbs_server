package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/lenchik-en/lbs_server/internal/config"
	"github.com/lenchik-en/lbs_server/internal/db"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

type App struct {
	locateDB    db.Finder
	externalDB  db.CellFinder
	updateDB    db.Inserter
	session     db.SessionManager
	cfg         *config.Config
	rateLimiter *ratelimit.RateLimiter
	wg          sync.WaitGroup
	refreshSrc  db.RefreshSource
	refreshDest db.RefreshDest
}

func (a *App) New(locate db.Finder, external db.CellFinder, update db.Inserter) {
	a.locateDB = locate
	a.externalDB = external
	a.updateDB = update
	a.session = db.NewLogger(locate.GetConnection())
}

func (a *App) Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	a.cfg = cfg
	a.rateLimiter = ratelimit.New()

	locateDB, err := db.NewLocateDB(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect locateDB: %v", err)
	}

	externalDB, err := db.NewExternalDB(cfg.EDBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect externalDB: %v", err)
	}

	updateDB, err := db.NewUpdateDB(cfg.UDBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect updateDB: %v", err)
	}

	a.New(locateDB, externalDB, updateDB)
	a.refreshSrc = updateDB
	a.refreshDest = locateDB

	return nil
}

// StartBackground запускает фоновые задачи (авто-refresh).
// Принимает ctx сервера — при его отмене задачи останавливаются.
func (a *App) StartBackground(ctx context.Context) {
	a.startRefreshLoop(ctx)
}

// Addr возвращает адрес из конфига для запуска HTTP-сервера.
func (a *App) Addr() string {
	return a.cfg.HTTPAddr
}

// Shutdown ожидает завершения фоновых горутин, закрывает соединения с БД и останавливает rate limiter.
func (a *App) Shutdown() {
	a.wg.Wait()
	if a.rateLimiter != nil {
		a.rateLimiter.Stop()
	}
	if a.locateDB != nil {
		a.locateDB.GetConnection().Close()
	}
	if a.externalDB != nil {
		a.externalDB.GetConnection().Close()
	}
	if a.updateDB != nil {
		a.updateDB.GetConnection().Close()
	}
}

func (a *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// checkRateLimit enforces the minimum-period rate limits for the given endpoint.
// endpoint is "locate" or "update".
// apiKey comes from the X-Api-Key request header; sessionUUID from the request body.
// Writes HTTP 429 and returns false if any limit is exceeded.
func (a *App) checkRateLimit(w http.ResponseWriter, endpoint, apiKey, sessionUUID string) bool {
	var keyPeriod, uuidPeriod, keyUUIDPeriod int
	switch endpoint {
	case "locate":
		keyPeriod = a.cfg.LocateMinPeriodKey
		uuidPeriod = a.cfg.LocateMinPeriodUUID
		keyUUIDPeriod = a.cfg.LocateMinPeriodKeyUUID
	case "update":
		keyPeriod = a.cfg.UpdateMinPeriodKey
		uuidPeriod = a.cfg.UpdateMinPeriodUUID
		keyUUIDPeriod = a.cfg.UpdateMinPeriodKeyUUID
	}

	if apiKey != "" && !a.rateLimiter.Allow(endpoint+":key:"+apiKey, keyPeriod) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return false
	}
	if sessionUUID != "" && !a.rateLimiter.Allow(endpoint+":uuid:"+sessionUUID, uuidPeriod) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return false
	}
	if apiKey != "" && sessionUUID != "" && !a.rateLimiter.Allow(endpoint+":key+uuid:"+apiKey+":"+sessionUUID, keyUUIDPeriod) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return false
	}
	return true
}
