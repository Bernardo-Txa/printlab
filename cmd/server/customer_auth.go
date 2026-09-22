package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type customerAuthService interface {
	Available() bool
	SignUp(context.Context, customerauth.SignUpInput) error
	Login(context.Context, http.ResponseWriter, string, string) error
	RecoverPassword(context.Context, string, string) error
	CompleteCallbackCode(context.Context, http.ResponseWriter, string) error
	CompleteCallbackSession(context.Context, http.ResponseWriter, customerauth.AuthSession) error
	UpdatePassword(context.Context, http.ResponseWriter, *http.Request, string) error
	Logout(context.Context, http.ResponseWriter, *http.Request)
	ResolveSession(context.Context, http.ResponseWriter, *http.Request) (customerauth.Profile, error)
	RedirectURL(*http.Request, string) string
}

func signupPageHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		renderHTML(w, r, http.StatusOK, templates.SignupPage(customerauth.SignupForm{}))
	}
}

func signupHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		if err := r.ParseForm(); err != nil {
			writeInvalidBodyError(w, "invalid signup request", err)
			return
		}
		input, fieldErrors := customerauth.ValidateSignup(r.PostFormValue("name"), r.PostFormValue("email"), r.PostFormValue("password"), r.PostFormValue("confirm_password"))
		form := customerauth.SignupForm{Name: input.Name, Email: input.Email, Errors: fieldErrors}
		if fieldErrors.Any() {
			renderHTML(w, r, http.StatusBadRequest, templates.SignupPage(form))
			return
		}
		input.RedirectURL = service.RedirectURL(r, "/auth/callback")
		if err := service.SignUp(r.Context(), input); err != nil {
			form.Message = signupErrorMessage(err)
			renderHTML(w, r, customerAuthStatus(err), templates.SignupPage(form))
			return
		}
		form.Message = customerauth.MessageSignupConfirmation
		renderHTML(w, r, http.StatusOK, templates.SignupPage(form))
	}
}

func loginPageHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		next, _ := customerauth.SafeRedirectPath(r.URL.Query().Get("next"), "/conta")
		message := ""
		if r.URL.Query().Get("senha") == "atualizada" {
			message = customerauth.MessagePasswordUpdated
		}
		renderHTML(w, r, http.StatusOK, templates.LoginPage(customerauth.LoginForm{Next: next, Message: message}))
	}
}

func loginHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		if err := r.ParseForm(); err != nil {
			writeInvalidBodyError(w, "invalid login request", err)
			return
		}
		email, password, next, fieldErrors := customerauth.ValidateLogin(r.PostFormValue("email"), r.PostFormValue("password"), r.PostFormValue("next"))
		form := customerauth.LoginForm{Email: email, Next: next, Errors: fieldErrors}
		if fieldErrors.Any() {
			renderHTML(w, r, http.StatusBadRequest, templates.LoginPage(form))
			return
		}
		if err := service.Login(r.Context(), w, email, password); err != nil {
			logCustomerAuthError(r.Context(), "customer_login_failed", err)
			form.Message = customerauth.MessageInvalidLogin
			if errors.Is(err, customerauth.ErrEmailNotConfirmed) {
				form.Message = customerauth.MessageEmailNotConfirmed
			}
			renderHTML(w, r, http.StatusUnauthorized, templates.LoginPage(form))
			return
		}
		http.Redirect(w, r, next, http.StatusSeeOther)
	}
}

func logoutHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if customerAuthAvailable(service) {
			service.Logout(r.Context(), w, r)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func recoveryPageHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		renderHTML(w, r, http.StatusOK, templates.RecoveryPage(customerauth.RecoveryForm{}))
	}
}

func recoveryHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		if err := r.ParseForm(); err != nil {
			writeInvalidBodyError(w, "invalid recovery request", err)
			return
		}
		email, fieldErrors := customerauth.ValidateRecovery(r.PostFormValue("email"))
		form := customerauth.RecoveryForm{Email: email, Errors: fieldErrors}
		if fieldErrors.Any() {
			renderHTML(w, r, http.StatusBadRequest, templates.RecoveryPage(form))
			return
		}
		if err := service.RecoverPassword(r.Context(), email, service.RedirectURL(r, "/auth/callback")); err != nil {
			logCustomerAuthError(r.Context(), "customer_recovery_failed", err)
		}
		form.Message = customerauth.MessageRecoverySent
		renderHTML(w, r, http.StatusOK, templates.RecoveryPage(form))
	}
}

func newPasswordPageHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if _, ok := requireCustomerSession(w, r, service); !ok {
			return
		}
		renderHTML(w, r, http.StatusOK, templates.NewPasswordPage(customerauth.NewPasswordForm{}))
	}
}

func newPasswordHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if _, ok := requireCustomerSession(w, r, service); !ok {
			return
		}
		if err := r.ParseForm(); err != nil {
			writeInvalidBodyError(w, "invalid password request", err)
			return
		}
		fieldErrors := customerauth.ValidateNewPassword(r.PostFormValue("password"), r.PostFormValue("confirm_password"))
		form := customerauth.NewPasswordForm{Errors: fieldErrors}
		if fieldErrors.Any() {
			renderHTML(w, r, http.StatusBadRequest, templates.NewPasswordPage(form))
			return
		}
		if err := service.UpdatePassword(r.Context(), w, r, r.PostFormValue("password")); err != nil {
			logCustomerAuthError(r.Context(), "customer_password_update_failed", err)
			form.Message = "Não foi possível atualizar a senha. Solicite um novo link e tente novamente."
			renderHTML(w, r, customerAuthStatus(err), templates.NewPasswordPage(form))
			return
		}
		http.Redirect(w, r, "/login?senha=atualizada", http.StatusSeeOther)
	}
}

type accountOrdersService interface {
	ListForCustomer(context.Context, string) ([]ordersdomain.AccountOrder, error)
}
type accountProfileService interface {
	Get(context.Context, string) (customerprofile.Profile, bool, error)
	Save(context.Context, customerprofile.Profile) error
}

func accountHandler(service customerAuthService, ordersService accountOrdersService, profiles accountProfileService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		profile, ok := requireCustomerSession(w, r, service)
		if !ok {
			return
		}
		customerOrders, err := ordersService.ListForCustomer(r.Context(), profile.ID)
		if err != nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		stored, _, _ := profiles.Get(r.Context(), profile.ID)
		renderHTML(w, r, http.StatusOK, templates.AccountPage(profile, customerOrders, stored))
	}
}

func accountSaveHandler(service customerAuthService, profiles accountProfileService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		profile, ok := requireCustomerSession(w, r, service)
		if !ok {
			return
		}
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid profile", 400)
			return
		}
		input := customers.CheckoutInput{FullName: r.PostFormValue("full_name"), Phone: r.PostFormValue("phone"), CPF: r.PostFormValue("cpf"), PostalCode: r.PostFormValue("postal_code"), Street: r.PostFormValue("street"), Number: r.PostFormValue("number"), Complement: r.PostFormValue("complement"), District: r.PostFormValue("district"), City: r.PostFormValue("city"), State: r.PostFormValue("state"), CountryCode: "BR"}
		_, values, errs := customers.NormalizeCheckoutInput(input)
		if errs.Any() {
			renderHTML(w, r, 400, templates.AccountPage(profile, nil, customerprofile.FromCheckout(profile.ID, values)))
			return
		}
		if err := profiles.Save(r.Context(), customerprofile.FromCheckout(profile.ID, values)); err != nil {
			renderHTML(w, r, 503, templates.AuthUnavailable())
			return
		}
		http.Redirect(w, r, "/conta?salvo=1", 303)
	}
}

func authCallbackHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
			return
		}
		if r.URL.Query().Get("confirmado") == "1" {
			if _, ok := requireCustomerSession(w, r, service); !ok {
				return
			}
			renderHTML(w, r, http.StatusOK, templates.AuthCallbackSuccessPage())
			return
		}
		if code := strings.TrimSpace(r.URL.Query().Get("code")); code != "" {
			if err := service.CompleteCallbackCode(r.Context(), w, code); err != nil {
				logCustomerAuthError(r.Context(), "customer_callback_code_failed", err)
				renderHTML(w, r, callbackErrorStatus(err), templates.AuthCallbackPage(customerauth.MessageCallbackInvalid))
				return
			}
			if callbackNextPath(r) == "/recuperar-senha/nova" {
				http.Redirect(w, r, "/recuperar-senha/nova", http.StatusSeeOther)
				return
			}
			renderHTML(w, r, http.StatusOK, templates.AuthCallbackSuccessPage())
			return
		}
		renderHTML(w, r, http.StatusOK, templates.AuthCallbackPage(""))
	}
}

type callbackSessionRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Next         string `json:"next"`
}

func authSessionHandler(service customerAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCustomerAuthPrivateHeaders(w)
		if !customerAuthAvailable(service) {
			http.Error(w, "auth unavailable", http.StatusServiceUnavailable)
			return
		}
		var payload callbackSessionRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&payload); err != nil {
			http.Error(w, "invalid auth request", http.StatusBadRequest)
			return
		}
		if payload.ExpiresIn <= 0 {
			payload.ExpiresIn = int(customerauth.DefaultAccessTTL.Seconds())
		}
		next, err := customerauth.SafeRedirectPath(payload.Next, "/conta")
		if err != nil {
			http.Error(w, "invalid redirect", http.StatusBadRequest)
			return
		}
		if err := service.CompleteCallbackSession(r.Context(), w, customerauth.AuthSession{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, ExpiresIn: payload.ExpiresIn}); err != nil {
			http.Error(w, "invalid auth request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"next": next})
	}
}

func callbackNextPath(r *http.Request) string {
	if r != nil && r.URL.Query().Get("type") == "recovery" {
		return "/recuperar-senha/nova"
	}
	return "/conta"
}

func callbackErrorStatus(err error) int {
	if errors.Is(err, customerauth.ErrUnavailable) || errors.Is(err, customerauth.ErrConfiguration) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func customerSessionMiddleware(next http.Handler, service customerAuthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !customerAuthAvailable(service) || skipCustomerSessionResolution(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		profile, err := service.ResolveSession(r.Context(), w, r)
		if err == nil {
			r = r.WithContext(customerauth.WithProfile(r.Context(), profile))
		} else if !errors.Is(err, customerauth.ErrUnauthenticated) {
			logCustomerAuthError(r.Context(), "customer_session_resolution_failed", err)
		}
		next.ServeHTTP(w, r)
	})
}

func skipCustomerSessionResolution(path string) bool {
	for _, prefix := range []string{"/static/", "/admin", "/webhooks/"} {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return path == "/health" || path == "/ready" || path == "/robots.txt" || path == "/sitemap.xml"
}

func requireCustomerSession(w http.ResponseWriter, r *http.Request, service customerAuthService) (customerauth.Profile, bool) {
	if !customerAuthAvailable(service) {
		renderHTML(w, r, http.StatusServiceUnavailable, templates.AuthUnavailable())
		return customerauth.Profile{}, false
	}
	if profile, ok := customerauth.ProfileFromContext(r.Context()); ok {
		return profile, true
	}
	profile, err := service.ResolveSession(r.Context(), w, r)
	if err != nil {
		loginURL := "/login?next=" + url.QueryEscape(r.URL.RequestURI())
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return customerauth.Profile{}, false
	}
	return profile, true
}

func customerAuthAvailable(service customerAuthService) bool {
	return service != nil && service.Available()
}

func setCustomerAuthPrivateHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
}

func signupErrorMessage(err error) string {
	if errors.Is(err, customerauth.ErrRateLimited) {
		return "Muitas tentativas. Aguarde alguns instantes e tente novamente."
	}
	return "Não foi possível criar a conta. Verifique os dados e tente novamente."
}

func customerAuthStatus(err error) int {
	if errors.Is(err, customerauth.ErrRateLimited) {
		return http.StatusTooManyRequests
	}
	if errors.Is(err, customerauth.ErrUnavailable) || errors.Is(err, customerauth.ErrConfiguration) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func logCustomerAuthError(ctx context.Context, event string, err error) {
	if err == nil {
		return
	}
	log.Printf("event=%s level=warning request_id=%s", event, requestIDFromContext(ctx))
}
