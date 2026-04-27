package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/service"
)

type RefreshHandler struct {
	svc *service.RefreshService
}

func NewRefreshHandler(svc *service.RefreshService) *RefreshHandler {
	return &RefreshHandler{svc: svc}
}

func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats, err := h.svc.Refresh(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("refresh failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("[WARN] failed to encode refresh stats: %v", err)
	}
}
