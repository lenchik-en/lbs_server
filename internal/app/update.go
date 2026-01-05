package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/api"
)

func (a *App) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] get /update request from client %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, cell := range req.Cell {
		if err := a.updateDB.InsertCell(r.Context(), cell); err != nil {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("Location is added to UpdateDB"); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("POST /update for Client %s is done", r.RemoteAddr)
}
