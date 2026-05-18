package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/service"
)

type UpdateHandler struct {
	svc *service.UpdateService
}

func NewUpdateHandler(svc *service.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] POST /update from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req model.UpdateRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
         	log.Printf("[ERROR] unmarshal error: %v, body: %s", err, string(rawBody))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.svc.Validate(&req); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.svc.Process(r.Context(), &req, rawBody); err != nil {
		http.Error(w, "Failed to insert to UpdateDB", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("Location is added to UpdateDB"); err != nil {
		log.Printf("[WARN] failed to encode update response: %v", err)
	}
}
