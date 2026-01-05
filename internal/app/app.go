package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/lenchik-en/lbs_server/internal/db"
)

type App struct {
	locateDB   *db.LocateDB
	externalDB *db.ExternalDB
	Session    *db.Session
}

func Run() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatalf("no path in DB_DSN")
	}

	edsn := os.Getenv("EDB_DSN")
	if edsn == "" {
		log.Fatalf("no path in EDB_DSN")
	}

	locateDB, err := db.NewLocateDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect locateDB: %v", err)
	}
	defer locateDB.DB.Close()

	externalDB, err := db.NewExternalDB(edsn)
	if err != nil {
		log.Fatalf("failed to connect externalDB: %v", err)
	}
	defer externalDB.DB.Close()

	app := &App{
		locateDB:   locateDB,
		externalDB: externalDB,
		Session:    db.NewLogger(locateDB.DB),
	}
	http.HandleFunc("/healthz", app.HandleHealth)

	http.HandleFunc("/locate", app.HandleLocate)

	//http.HandleFunc("/update", app.HandleUpdate)

	fmt.Println("Server listening on :8080...")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func (a *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
