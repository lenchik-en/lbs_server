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

	//todo: закинуть в Run
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", app.HandleHealth)
	mux.HandleFunc("/locate", app.HandleLocate)
	mux.HandleFunc("/update", app.HandleUpdate)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
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
	//todo: сделать фцнкцию app.SHutdown()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server stoppped")
}
