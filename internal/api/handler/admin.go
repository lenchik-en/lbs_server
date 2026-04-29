package handler

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/lenchik-en/lbs_server/internal/api/middleware"
	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/service"
)

//go:embed templates/*.html
var adminTemplates embed.FS

type AdminHandler struct {
	adminSvc   *service.AdminService
	apiKeySvc  *service.APIKeyService
	refreshSvc *service.RefreshService
}

func NewAdminHandler(adminSvc *service.AdminService, apiKeySvc *service.APIKeyService, refreshSvc *service.RefreshService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, apiKeySvc: apiKeySvc, refreshSvc: refreshSvc}
}

func (h *AdminHandler) render(w http.ResponseWriter, name string, data map[string]any) {
	data["LoggedIn"] = true
	t, err := template.New("").ParseFS(adminTemplates, "templates/layout.html", "templates/"+name)
	if err != nil {
		log.Printf("[WARN] parse %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("[WARN] render %s: %v", name, err)
	}
}

func (h *AdminHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// Уже залогинен?
	if c, err := r.Cookie(middleware.AdminSessionCookie); err == nil && h.adminSvc.ValidateSession(c.Value) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
		return
	}
	h.renderLogin(w, "")
}

func (h *AdminHandler) renderLogin(w http.ResponseWriter, errMsg string) {
	t, err := template.New("").ParseFS(adminTemplates, "templates/layout.html", "templates/login.html")
	if err != nil {
		log.Printf("[WARN] parse login: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "login.html", map[string]any{
		"Title": "Вход", "LoggedIn": false, "Error": errMsg,
	}); err != nil {
		log.Printf("[WARN] render login: %v", err)
	}
}

func (h *AdminHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	token, err := h.adminSvc.Login(login, password)
	if err != nil {
		h.renderLogin(w, "Неверный логин или пароль")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AdminSessionCookie,
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(8 * time.Hour),
	})
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(middleware.AdminSessionCookie); err == nil {
		h.adminSvc.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: middleware.AdminSessionCookie, Path: "/admin", MaxAge: -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.render(w, "dashboard.html", map[string]any{
		"Title":    "Настройки",
		"Settings": h.adminSvc.GetSettings(),
		"Flash":    r.URL.Query().Get("flash"),
	})
}

func (h *AdminHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	parseInt := func(key string) int { v, _ := strconv.Atoi(r.FormValue(key)); return v }

	h.adminSvc.UpdateSettings(service.Settings{
		RequireKeyLocate:    r.FormValue("require_key_locate") == "on",
		RequireKeyUpdate:    r.FormValue("require_key_update") == "on",
		RefreshPeriodMs:     parseInt("refresh_period_ms"),
		LocateMinPeriodKey:  parseInt("locate_min_period_key"),
		LocateMinPeriodAnon: parseInt("locate_min_period_anon"),
		UpdateMinPeriodKey:  parseInt("update_min_period_key"),
		UpdateMinPeriodAnon: parseInt("update_min_period_anon"),
	})
	http.Redirect(w, r, "/admin/?flash=Настройки+сохранены", http.StatusFound)
}

func (h *AdminHandler) RunRefresh(w http.ResponseWriter, r *http.Request) {
	stats, err := h.refreshSvc.Refresh(context.Background())
	if err != nil {
		log.Printf("[WARN] manual refresh: %v", err)
		http.Redirect(w, r, "/admin/?flash=Ошибка+refresh", http.StatusFound)
		return
	}
	h.render(w, "dashboard.html", map[string]any{
		"Title":         "Настройки",
		"Settings":      h.adminSvc.GetSettings(),
		"RefreshResult": stats,
	})
}

func (h *AdminHandler) KeysPage(w http.ResponseWriter, r *http.Request) {
	keys, err := h.apiKeySvc.ListAll(r.Context())
	if err != nil {
		http.Error(w, "failed to load keys", http.StatusInternalServerError)
		return
	}
	h.render(w, "keys.html", map[string]any{
		"Title": "API-ключи",
		"Keys":  keys,
		"Flash": r.URL.Query().Get("flash"),
	})
}

func (h *AdminHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	scope := model.KeyScope(r.FormValue("scope"))
	switch scope {
	case model.ScopeLocate, model.ScopeUpdate, model.ScopeBoth:
	default:
		http.Redirect(w, r, "/admin/keys", http.StatusFound)
		return
	}

	key, err := h.apiKeySvc.Generate(r.Context(), scope)
	if err != nil {
		log.Printf("[WARN] create key: %v", err)
		http.Redirect(w, r, "/admin/keys", http.StatusFound)
		return
	}

	keys, _ := h.apiKeySvc.ListAll(r.Context())
	h.render(w, "keys.html", map[string]any{
		"Title":  "API-ключи",
		"Keys":   keys,
		"NewKey": key.Key,
	})
}

func (h *AdminHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	if key == "" {
		http.Redirect(w, r, "/admin/keys", http.StatusFound)
		return
	}
	if err := h.apiKeySvc.Revoke(r.Context(), key); err != nil {
		log.Printf("[WARN] revoke key: %v", err)
	}
	http.Redirect(w, r, "/admin/keys?flash=Ключ+отозван", http.StatusFound)
}
