package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/service"
)

type LocateHandler struct {
	svc *service.LocateService
}

func NewLocateHandler(svc *service.LocateService) *LocateHandler {
	return &LocateHandler{svc: svc}
}

func (h *LocateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] POST /locate from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req model.LocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.FindLocation(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find location: %v", err), http.StatusInternalServerError)
		return
	}
	if resp == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[WARN] failed to encode locate response: %v", err)
	}
}
