package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/service"
)

type APIKeyHandler struct {
	svc *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

// POST /admin/keys  body: {"scope":"locate"|"update"|"both"}
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope model.KeyScope `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	switch req.Scope {
	case model.ScopeLocate, model.ScopeUpdate, model.ScopeBoth:
	default:
		http.Error(w, fmt.Sprintf("invalid scope %q", req.Scope), http.StatusBadRequest)
		return
	}

	key, err := h.svc.Generate(r.Context(), req.Scope)
	if err != nil {
		http.Error(w, "failed to generate key", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(key); err != nil {
		log.Printf("[WARN] encode key: %v", err)
	}
}

// DELETE /admin/keys?key=<key>
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	if err := h.svc.Revoke(r.Context(), key); err != nil {
		http.Error(w, "failed to revoke key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /admin/keys
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.svc.ListAll(r.Context())
	if err != nil {
		http.Error(w, "failed to list keys", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		log.Printf("[WARN] encode keys: %v", err)
	}
}
