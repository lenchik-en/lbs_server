package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/lenchik-en/lbs_server/internal/api"
	"github.com/lenchik-en/lbs_server/internal/config"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
	"github.com/lenchik-en/lbs_server/internal/repository/postgres"
	"github.com/lenchik-en/lbs_server/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- репозитории ---
	locateRepo, err := postgres.NewLocateRepo(cfg.DBDSN)
	if err != nil {
		log.Fatalf("locateRepo: %v", err)
	}
	externalRepo, err := postgres.NewExternalRepo(cfg.EDBDSN)
	if err != nil {
		log.Fatalf("externalRepo: %v", err)
	}
	updateRepo, err := postgres.NewUpdateRepo(cfg.UDBDSN)
	if err != nil {
		log.Fatalf("updateRepo: %v", err)
	}

	// SessionRepo живёт в LocateDB (там же sessions/session_points)
	sessionRepo := postgres.NewSessionRepo(locateRepo.DB())
	apiKeyRepo := postgres.NewAPIKeyRepo(locateRepo.DB())

	// --- сервисы ---
	wg := &service.WaitGroupAdapter{}

	locateSvc := service.NewLocateService(locateRepo, externalRepo, updateRepo, sessionRepo, wg)
	updateSvc := service.NewUpdateService(updateRepo, sessionRepo)
	refreshSvc := service.NewRefreshService(updateRepo, locateRepo)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	adminSvc := service.NewAdminService(cfg.AdminLogin, cfg.AdminPasswordHash, service.Settings{
		RequireKeyLocate: cfg.RequireKeyLocate,
		RequireKeyUpdate: cfg.RequireKeyUpdate,
		RefreshPeriodMs:  cfg.RefreshPeriodMs,
	})

	// --- авто-refresh (период берём из AdminService — он может меняться через UI) ---
	refreshSvc.StartAutoRefresh(ctx, cfg.RefreshPeriodMs)

	// --- HTTP ---
	limiter := ratelimit.New()
	mux := api.NewMux(adminSvc, limiter, locateSvc, updateSvc, refreshSvc, apiKeySvc)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}

	wg.Wait()
	limiter.Stop()
	log.Println("Server stopped")
}
