package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/lenchik-en/lbs_server/internal/api"
	"github.com/lenchik-en/lbs_server/internal/db"
)

type App struct {
	DB     *sql.DB
	Logger *db.Logger
}

func (a *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *App) HandleLocate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.LocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	//TODO: уточнить нужно ли присваивать, если не получен от клиента
	if req.SessionUUID == "" {
		req.SessionUUID = uuid.New().String()
	}
	log.Printf("SessionUUID: %s", req.SessionUUID)

	logger := a.Logger

	_ = logger.CreateSessionIfNotExists(r.Context(), req.SessionUUID)

	log.Printf("Convert to OpenCellID...")
	openCellRequest := api.ConvertLocateToOpenCell(req)

	var best *api.OpenCellResponse
	for _, r := range openCellRequest {
		resp, err := api.Query(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("error from OpenCellID: %v", err), http.StatusInternalServerError)
			continue
		}
		if resp.Status == "ok" {
			best = resp
			break
		}
	}

	if best == nil {
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	_ = logger.SavePoint(r.Context(), req.SessionUUID, best.Lat, best.Lon, best.Accuracy, "opencell", req, best)

	loc := api.ConvertOpenCellToLocation(best)

	out := map[string]any{
		"sessionUUID": req.SessionUUID,
		"location":    loc,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (a *App) HandleLocateOpenCell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.OpenCellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resp, err := api.Query(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error from OpenCellID: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Run(dab *db.LocateDB) {
	app := &App{
		DB:     dab.DB,
		Logger: db.NewLogger(dab.DB),
	}
	http.HandleFunc("/healthz", app.HandleHealth)

	http.HandleFunc("/locate", app.HandleLocate)

	http.HandleFunc("/locate/test", app.HandleLocateOpenCell)

	fmt.Println("Server listening on :8080...")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
