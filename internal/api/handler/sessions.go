package handler

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/lenchik-en/lbs_server/internal/service"
)

type SessionsHandler struct {
	svc *service.SessionAdminService
}

func NewSessionsHandler(svc *service.SessionAdminService) *SessionsHandler {
	return &SessionsHandler{svc: svc}
}

// Route диспетчеризирует все запросы под /admin/sessions/.
func (h *SessionsHandler) Route(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/admin/sessions/")

	// POST /admin/sessions/{uuid}/refresh
	if strings.HasSuffix(rest, "/refresh") && r.Method == http.MethodPost {
		uuid := strings.TrimSuffix(rest, "/refresh")
		h.refreshSession(w, r, uuid)
		return
	}

	// PATCH /admin/sessions/{uuid}/points/{id}
	if i := strings.Index(rest, "/points/"); i >= 0 && r.Method == http.MethodPatch {
		uuid := rest[:i]
		idStr := rest[i+len("/points/"):]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid point id", http.StatusBadRequest)
			return
		}
		h.updatePoint(w, r, uuid, id)
		return
	}

	// GET /admin/sessions/{uuid}
	h.DetailPage(w, r)
}

func (h *SessionsHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	sessions, total, err := h.svc.List(r.Context(), page)
	if err != nil {
		log.Printf("[WARN] sessions list: %v", err)
		http.Error(w, "failed to load sessions", http.StatusInternalServerError)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(service.SessionPageSize)))

	h.render(w, "sessions.html", map[string]any{
		"Title":      "Маршруты",
		"Sessions":   sessions,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
	})
}

func (h *SessionsHandler) DetailPage(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.Path, "/admin/sessions/")
	if uuid == "" {
		http.Redirect(w, r, "/admin/sessions", http.StatusFound)
		return
	}

	points, err := h.svc.GetPoints(r.Context(), uuid)
	if err != nil {
		log.Printf("[WARN] session detail %s: %v", uuid, err)
		http.Error(w, "failed to load session", http.StatusInternalServerError)
		return
	}

	flash := r.URL.Query().Get("flash")

	h.render(w, "session_detail.html", map[string]any{
		"Title":       "Маршрут " + uuid[:8] + "…",
		"SessionUUID": uuid,
		"Points":      points,
		"Flash":       flash,
	})
}

func (h *SessionsHandler) updatePoint(w http.ResponseWriter, r *http.Request, uuid string, id int64) {
	var body struct {
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Accuracy float64 `json:"accuracy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Lat < -90 || body.Lat > 90 || body.Lon < -180 || body.Lon > 180 {
		http.Error(w, "coordinates out of range", http.StatusBadRequest)
		return
	}
	if body.Accuracy < 0 {
		http.Error(w, "accuracy must be non-negative", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdatePoint(r.Context(), id, body.Lat, body.Lon, body.Accuracy); err != nil {
		log.Printf("[WARN] update point %d: %v", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (h *SessionsHandler) refreshSession(w http.ResponseWriter, r *http.Request, uuid string) {
	stats, err := h.svc.RefreshFromSession(r.Context(), uuid)
	if err != nil {
		log.Printf("[WARN] refresh session %s: %v", uuid, err)
		http.Error(w, "refresh failed", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] session refresh %s: cells=%d wifis=%d ips=%d",
		uuid, stats.CellsUpserted, stats.WifisUpserted, stats.IPsUpserted)

	http.Redirect(w, r,
		"/admin/sessions/"+uuid+"?flash=Refresh+выполнен:+ячеек+"+
			strconv.Itoa(stats.CellsUpserted)+",+wifi+"+
			strconv.Itoa(stats.WifisUpserted),
		http.StatusSeeOther,
	)
}

func (h *SessionsHandler) render(w http.ResponseWriter, name string, data map[string]any) {
	data["LoggedIn"] = true
	t, err := parseAdminTemplates(name)
	if err != nil {
		log.Printf("[WARN] sessions parse %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("[WARN] sessions render %s: %v", name, err)
	}
}
