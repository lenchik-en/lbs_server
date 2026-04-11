package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/lenchik-en/lbs_server/internal/app"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// godotenv используется для локальной разработки без Docker.
	// В контейнере переменные уже инжектируются через env_file в docker-compose,
	// поэтому ошибку отсутствия файла .env игнорируем.
	//if err := godotenv.Load(".env"); err != nil {
	//	log.Println("no .env file found, using environment variables")
	//}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app := &app.App{}

	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}

	if err := app.LoadDemoCells(ctx); err != nil {
		log.Printf("[WARN] demo cells not loaded: %v", err)
	}

	app.StartBackground(ctx)

	//todo: закинуть в Run
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.HandleHealth)
	mux.HandleFunc("/locate", app.HandleLocate)
	mux.HandleFunc("/update", app.HandleUpdate)
	mux.HandleFunc("/refresh", app.HandleRefresh)
	mux.HandleFunc("/demo/cells", app.HandleDemoCells)

	server := &http.Server{
		Addr:         app.Addr(),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("could not listen on :8080: %v\n", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	app.Shutdown()
	log.Println("Server stopped")
}
