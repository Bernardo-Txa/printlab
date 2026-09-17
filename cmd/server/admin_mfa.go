package main

import (
	"context"
	"errors"
	"net/http"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

func adminMFAPageHandler(service adminPanelService, setup bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !adminAvailable(service) {
			adminMFATerminalError(w, r, service, admindomain.ErrUnavailable)
			return
		}
		// HEAD must not create or remove an enrollment.
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		page, err := service.MFAPage(r.Context(), r, setup)
		if err != nil {
			adminMFATerminalError(w, r, service, err)
			return
		}
		if page.Setup != setup {
			path := "/admin/mfa/challenge"
			if page.Setup {
				path = "/admin/mfa/setup"
			}
			http.Redirect(w, r, path, http.StatusSeeOther)
			return
		}
		renderHTML(w, r, http.StatusOK, templates.AdminMFA(page))
	}
}

func adminMFAVerifyHandler(service adminPanelService, siteURL string, setup bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validAdminMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !adminAvailable(service) {
			adminMFATerminalError(w, r, service, admindomain.ErrUnavailable)
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			service.ClearPendingCookie(w)
			writeInvalidBodyError(w, "invalid admin request", err)
			return
		}
		result, page, err := service.CompleteMFA(r.Context(), r, setup, r.PostFormValue("factor_id"), r.PostFormValue("code"))
		if err != nil {
			if (errors.Is(err, admindomain.ErrMFAInvalidCode) || errors.Is(err, admindomain.ErrAuthRateLimited)) && (page.FactorID != "" || len(page.Factors) > 0) {
				status, message := adminAuthError(r.Context(), err)
				page.Message = message
				renderHTML(w, r, status, templates.AdminMFA(page))
				return
			}
			adminMFATerminalError(w, r, service, err)
			return
		}
		service.WriteCookie(w, result.Token, result.ExpiresAt)
		service.ClearPendingCookie(w)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func adminMFACancelHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validAdminMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if service != nil {
			service.ClearPendingCookie(w)
		}
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func adminMFATerminalError(w http.ResponseWriter, r *http.Request, service adminPanelService, err error) {
	if service != nil {
		service.ClearPendingCookie(w)
	}
	if errors.Is(err, admindomain.ErrUnauthenticated) || errors.Is(err, admindomain.ErrMFAFactor) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	status, message := adminAuthError(r.Context(), err)
	renderHTML(w, r, status, templates.AdminLogin(message))
}

func adminAuthError(ctx context.Context, err error) (int, string) {
	switch {
	case errors.Is(err, admindomain.ErrInvalidCredentials), errors.Is(err, admindomain.ErrAuthRejected), errors.Is(err, admindomain.ErrUnauthenticated):
		logOperationalEvent(ctx, "admin_auth_rejected", "reason=invalid_credentials")
		return http.StatusUnauthorized, admindomain.InvalidCredentialsMessage
	case errors.Is(err, admindomain.ErrMFAInvalidCode):
		logOperationalEvent(ctx, "admin_mfa_invalid_code", "reason=invalid_code")
		return http.StatusUnauthorized, "Código inválido ou expirado."
	case errors.Is(err, admindomain.ErrAuthRateLimited):
		logOperationalEvent(ctx, "admin_mfa_rate_limited", "reason=provider_rate_limited")
		return http.StatusTooManyRequests, "Muitas tentativas. Aguarde antes de tentar novamente."
	case errors.Is(err, admindomain.ErrAuthConfiguration):
		logOperationalEvent(ctx, "admin_auth_rejected", "reason=configuration_invalid")
	case errors.Is(err, admindomain.ErrAuthInvalidResponse):
		logOperationalEvent(ctx, "admin_mfa_provider_unavailable", "reason=invalid_response")
	default:
		logOperationalEvent(ctx, "admin_mfa_provider_unavailable", "reason=provider_unavailable")
	}
	return http.StatusServiceUnavailable, "Acesso temporariamente indisponível. Tente novamente mais tarde."
}
