package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/customerauth"
)

func TestSignupValidCreatesSupabaseUser(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"name": {"Maria Cliente"}, "email": {"maria@example.com"}, "password": {"senha-segura"}, "confirm_password": {"senha-segura"}}
	req := httptest.NewRequest(http.MethodPost, "/cadastro", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !service.signupCalled || service.signupInput.Email != "maria@example.com" || service.signupInput.Name != "Maria Cliente" {
		t.Fatalf("expected signup input, got %#v", service.signupInput)
	}
	if !strings.Contains(rec.Body.String(), customerauth.MessageSignupConfirmation) {
		t.Fatal("expected confirmation message")
	}
}

func TestSignupValidationRejectsMismatchedPassword(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"name": {"Maria"}, "email": {"maria@example.com"}, "password": {"senha-segura"}, "confirm_password": {"outra"}}
	req := httptest.NewRequest(http.MethodPost, "/cadastro", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if service.signupCalled {
		t.Fatal("expected invalid form not to call Supabase")
	}
	if !strings.Contains(rec.Body.String(), "As senhas não coincidem.") {
		t.Fatal("expected password mismatch message")
	}
}

func TestLoginValidRedirectsToSafeNext(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-segura"}, "next": {"/conta"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta" {
		t.Fatalf("expected redirect to account, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.loginCalled || service.loginEmail != "cliente@example.com" || service.loginPassword != "senha-segura" {
		t.Fatalf("expected login call, got %#v", service)
	}
}

func TestLoginInvalidShowsGenericMessage(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, loginErr: customerauth.ErrRejected}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-errada"}, "next": {"/conta"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, customerauth.MessageInvalidLogin) || strings.Contains(body, "senha-errada") {
		t.Fatal("expected generic invalid login message without password")
	}
}

func TestLoginRejectsOpenRedirect(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-segura"}, "next": {"https://evil.example"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta" {
		t.Fatalf("expected fallback redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestRecoveryDoesNotEnumerateUsers(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, recoveryErr: customerauth.ErrRejected}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"desconhecido@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/recuperar-senha", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), customerauth.MessageRecoverySent) {
		t.Fatal("expected generic recovery message")
	}
}

func TestAuthSessionCallbackSetsSupabaseSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/auth/session", strings.NewReader(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"next":"/conta"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !service.callbackCalled {
		t.Fatal("expected callback session completion")
	}
}

func TestAccountRequiresAuthentication(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login?next=") {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAccountRendersAuthenticatedUser(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, profile: customerauth.Profile{ID: "user-1", Name: "Maria Cliente", Email: "maria@example.com"}}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Minha conta", "Maria Cliente", "maria@example.com", "Conta ativa"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected account page to contain %q", expected)
		}
	}
}

func TestLogoutClearsSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("expected redirect home, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.logoutCalled {
		t.Fatal("expected logout call")
	}
}

func TestGuestCheckoutStillWorksWithoutCustomerSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/carrinho", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected guest cart to keep working, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Entrar para comprar") {
		t.Fatal("expected guest checkout path not to require login")
	}
}

func newCustomerAuthTestHandler(service customerAuthService) http.Handler {
	return newHandlerWithServicesAndOrdersAndCustomerAuthAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, service, "https://printlab.test", "https://supabase.test")
}

type fakeCustomerAuthService struct {
	available bool

	signupCalled bool
	signupInput  customerauth.SignUpInput
	signupErr    error

	loginCalled   bool
	loginEmail    string
	loginPassword string
	loginErr      error

	recoveryCalled bool
	recoveryEmail  string
	recoveryErr    error

	callbackCalled bool
	callbackErr    error

	updateErr error

	logoutCalled bool

	profile    customerauth.Profile
	resolveErr error
}

func (s *fakeCustomerAuthService) Available() bool { return s != nil && s.available }
func (s *fakeCustomerAuthService) SignUp(_ context.Context, input customerauth.SignUpInput) error {
	s.signupCalled = true
	s.signupInput = input
	return s.signupErr
}
func (s *fakeCustomerAuthService) Login(_ context.Context, _ http.ResponseWriter, email string, password string) error {
	s.loginCalled = true
	s.loginEmail = email
	s.loginPassword = password
	return s.loginErr
}
func (s *fakeCustomerAuthService) RecoverPassword(_ context.Context, email string, _ string) error {
	s.recoveryCalled = true
	s.recoveryEmail = email
	return s.recoveryErr
}
func (s *fakeCustomerAuthService) CompleteCallbackSession(_ context.Context, _ http.ResponseWriter, _ customerauth.AuthSession) error {
	s.callbackCalled = true
	return s.callbackErr
}
func (s *fakeCustomerAuthService) UpdatePassword(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ string) error {
	return s.updateErr
}
func (s *fakeCustomerAuthService) Logout(_ context.Context, _ http.ResponseWriter, _ *http.Request) {
	s.logoutCalled = true
}
func (s *fakeCustomerAuthService) ResolveSession(_ context.Context, _ http.ResponseWriter, _ *http.Request) (customerauth.Profile, error) {
	if s.resolveErr != nil {
		return customerauth.Profile{}, s.resolveErr
	}
	if s.profile.ID == "" {
		return customerauth.Profile{}, customerauth.ErrUnauthenticated
	}
	return s.profile, nil
}
func (s *fakeCustomerAuthService) RedirectURL(_ *http.Request, path string) string {
	return "https://printlab.test" + path
}
