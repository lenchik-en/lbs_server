package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/api"
	"github.com/lenchik-en/lbs_server/internal/db"
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func HandleLocate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.LocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"debuq": req,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Run(db *db.LocateDB) {
	http.HandleFunc("/healthz", HandleHealth)

	http.HandleFunc("/locate", HandleLocate)

	fmt.Println("Server listening on :8080...")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
