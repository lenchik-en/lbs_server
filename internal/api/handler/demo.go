package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

type DemoHandler struct {
	cells []model.LocateRequest
}

func NewDemoHandler(ctx interface{ Done() <-chan struct{} }, repo port.ExternalRepository, limit int) *DemoHandler {
	return &DemoHandler{}
}

func (h *DemoHandler) Load(cells []model.LocateRequest) {
	h.cells = cells
}

func (h *DemoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.cells) == 0 {
		http.Error(w, "demo cells not loaded", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.cells); err != nil {
		log.Printf("[WARN] encode demo cells: %v", err)
	}
}
