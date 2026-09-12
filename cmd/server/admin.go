package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type adminPanelService interface {
	Available() bool
	Login(ctx context.Context, email string, password string) (admindomain.LoginResult, error)
	ResolveSession(ctx context.Context, r *http.Request) (admindomain.Session, error)
	Logout(ctx context.Context, r *http.Request) error
	Dashboard(ctx context.Context) (admindomain.Dashboard, error)
	WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time)
	ClearCookie(w http.ResponseWriter)
}

func adminLoginPageHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !adminAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminLogin(""))
	}
}

func adminLoginHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !adminAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}

		result, err := service.Login(r.Context(), strings.TrimSpace(r.PostFormValue("email")), r.PostFormValue("password"))
		if err != nil {
			log.Print("admin login failed")
			renderHTML(w, r, http.StatusUnauthorized, templates.AdminLogin(admindomain.InvalidCredentialsMessage))
			return
		}

		log.Print("admin login succeeded")
		service.WriteCookie(w, result.Token, result.ExpiresAt)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func adminDashboardHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !requireAdminSession(w, r, service) {
			return
		}

		dashboard, err := service.Dashboard(r.Context())
		if err != nil {
			log.Print("admin repository error")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminDashboard(dashboard))
	}
}

func adminLogoutHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if adminAvailable(service) {
			if err := service.Logout(r.Context(), r); err != nil && !errors.Is(err, admindomain.ErrUnauthenticated) {
				log.Print("admin repository error")
			}
			service.ClearCookie(w)
		}

		log.Print("admin logout")
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func adminProtectedNotFoundHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !requireAdminSession(w, r, service) {
			return
		}

		http.NotFound(w, r)
	}
}

func requireAdminSession(w http.ResponseWriter, r *http.Request, service adminPanelService) bool {
	if !adminAvailable(service) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}

	if _, err := service.ResolveSession(r.Context(), r); err != nil {
		if errors.Is(err, admindomain.ErrSessionExpired) {
			log.Print("admin session expired")
		}
		service.ClearCookie(w)
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}

	return true
}

func adminAvailable(service adminPanelService) bool {
	return service != nil && service.Available()
}

func setAdminPrivateHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", checkoutPrivateCacheControl)
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Referrer-Policy", "same-origin")
}
