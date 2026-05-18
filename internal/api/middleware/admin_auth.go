package middleware

import (
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/service"
)

const AdminSessionCookie = "admin_session"

// AdminAuth проверяет сессионную куку. При неверной сессии редиректит на /admin/login.
func AdminAuth(svc *service.AdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AdminSessionCookie)
			if err != nil || !svc.ValidateSession(cookie.Value) {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AdminAPIAuth проверяет сессионную куку. При неверной сессии возвращает 401 (для JSON API).
func AdminAPIAuth(svc *service.AdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AdminSessionCookie)
			if err != nil || !svc.ValidateSession(cookie.Value) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
