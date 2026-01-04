package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/lenchik-en/lbs_server/internal/api"
	"github.com/lenchik-en/lbs_server/internal/db"
)

type App struct {
	locateDB   *db.LocateDB
	externalDB *db.ExternalDB
	Session    *db.Session
}

func (a *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *App) HandleLocate(w http.ResponseWriter, r *http.Request) {
	log.Printf("Get /locate request from Client %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.LocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	//TODO: уточнить нужно ли присваивать, если не получен от клиента(или генерирует со стороны клиента)
	if req.SessionUUID == "" {
		req.SessionUUID = uuid.New().String()
	}
	log.Printf("SessionUUID: %s", req.SessionUUID)

	session := a.Session
	err := session.CreateSessionIfNotExists(r.Context(), req.SessionUUID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add sessionUUID to the database: %v", err), http.StatusInternalServerError)
		//return
	}

	var location *db.Location
	for _, cell := range req.Cell {
		//1. Looking at LocateDB
		loc, err := a.locateDB.FindLTE(r.Context(), cell.LTE)
		if err != nil {
			http.Error(w, fmt.Sprintf("locateDB error: %v", err), http.StatusInternalServerError)
			return
		}

		//2. If not in LocateDB, then looking at ExternalDB
		if loc == nil {
			loc, err = a.externalDB.FindLTE(r.Context(), cell.LTE)
			if err != nil {
				http.Error(w, fmt.Sprintf("externalDB error: %v", err), http.StatusInternalServerError)
				return
			}

			//3. if found, then save it to UpdateDB(TODO: or LocateDB)
			if loc != nil {
				//_ = a.locateDB.SaveLTE
			}
		}

		if loc != nil {
			location = loc
			break
		}

	}

	if location == nil {
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	_ = session.SavePoint(r.Context(), req.SessionUUID, location.Point.Lat, location.Point.Lon, location.Accuracy, "opencell", req, location)

	out := map[string]any{
		"sessionUUID": req.SessionUUID,
		"location":    location,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
	log.Printf("POST /locate for Client %s is done", r.RemoteAddr)
}

func Run() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatalf("no path in DB_DSN")
	}

	edsn := os.Getenv("EDB_DSN")
	if dsn == "" {
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
