package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
)

func TestAdminLoginPage(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, &fakeAdminPanelService{available: true}, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	body := rec.Body.String()
	if !strings.Contains(body, "Acesso administrativo") || !strings.Contains(body, `autocomplete="current-password"`) {
		t.Fatal("expected admin login form")
	}
}

func TestAdminLoginUnavailableWhenConfigMissing(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, nil, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	if strings.Contains(rec.Body.String(), "SUPABASE") || strings.Contains(rec.Body.String(), "ADMIN_SUPABASE_USER_ID") {
		t.Fatal("expected unavailable page not to reveal missing environment variables")
	}
}

func TestAdminLoginValidRedirectsAndSetsCookie(t *testing.T) {
	expiresAt := time.Now().Add(admindomain.SessionTTL)
	service := &fakeAdminPanelService{
		available: true,
		loginResult: admindomain.LoginResult{
			Token:     mustAdminToken(t),
			ExpiresAt: expiresAt,
		},
	}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	form := url.Values{"email": {"admin@example.com"}, "password": {"correct-password"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin" {
		t.Fatalf("expected redirect to /admin, got %q", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != admindomain.CookieName {
		t.Fatalf("expected admin session cookie, got %#v", cookies)
	}
	if cookies[0].Path != "/admin" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected safe admin cookie attributes, got %#v", cookies[0])
	}
	if service.loginEmail != "admin@example.com" || service.loginPassword != "correct-password" {
		t.Fatalf("expected login credentials to be forwarded only to service, got %q/%q", service.loginEmail, service.loginPassword)
	}
}

func TestAdminLoginInvalidCredentialsShowsGenericMessageAndNoCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true, loginErr: admindomain.ErrInvalidCredentials}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	form := url.Values{"email": {"other@example.com"}, "password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), admindomain.InvalidCredentialsMessage) {
		t.Fatal("expected generic invalid credentials message")
	}
	if strings.Contains(rec.Body.String(), "não autorizado") || strings.Contains(rec.Body.String(), "wrong-password") || len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected no authorization details, password, or cookie")
	}
}

func TestAdminLoginRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader("email=admin@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.loginCalled {
		t.Fatal("expected cross-site login not to call service")
	}
}

func TestAdminDashboardRedirectsWithoutSession(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrUnauthenticated}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
	if !service.resolveCalled {
		t.Fatal("expected session guard")
	}
}

func TestAdminDashboardRedirectsToLoginWhenAdminUnavailable(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, nil, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
}

func TestAdminDashboardRendersForValidSession(t *testing.T) {
	service := &fakeAdminPanelService{
		available: true,
		dashboard: admindomain.Dashboard{
			PendingPayment:        1,
			PaidWaitingProduction: 2,
			InProduction:          3,
			WaitingShipment:       4,
		},
	}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	body := rec.Body.String()
	for _, expected := range []string{"Aguardando pagamento", ">1<", "Pagos aguardando produção", ">2<", "Em produção", ">3<", "Aguardando envio", ">4<"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected dashboard body to contain %q", expected)
		}
	}
	for _, forbidden := range []string{"CPF", "transaction_nsu", "checkout_url", "admin@example.com", "Rua "} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected dashboard not to expose %q", forbidden)
		}
	}
}

func TestAdminDashboardExpiredSessionRedirectsAndClearsCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrSessionExpired}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected expired admin cookie to be cleared, got %#v", rec.Result().Cookies())
	}
}

func TestAdminDashboardRandomCookieRedirects(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrUnauthenticated}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: "random"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
}

func TestAdminLogoutPostValidatesOriginDeletesSessionAndClearsCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.Header.Set("Origin", "https://printlab.test")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.logoutCalled {
		t.Fatal("expected logout to call service")
	}
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected admin cookie to be cleared, got %#v", rec.Result().Cookies())
	}
}

func TestAdminLogoutRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.logoutCalled {
		t.Fatal("expected cross-site logout not to call service")
	}
}

func TestAdminLogoutGetDoesNotLogout(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected method not allowed, got %d", rec.Code)
	}
	if service.logoutCalled {
		t.Fatal("expected GET logout not to call service")
	}
}

func newTestHandlerWithAdmin(t *testing.T, adminPanel adminPanelService, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServicesAndOrders(db, nil, nil, nil, nil, nil, nil, nil, nil, adminPanel, siteURL)
}

func assertAdminHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	if robots := rec.Header().Get("X-Robots-Tag"); !strings.Contains(robots, "noindex") || !strings.Contains(robots, "nofollow") || !strings.Contains(robots, "noarchive") {
		t.Fatalf("expected noindex robots header, got %q", robots)
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected no-referrer policy, got %q", rec.Header().Get("Referrer-Policy"))
	}
}

func assertAdminRedirectToLogin(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("expected redirect to login, got %q", rec.Header().Get("Location"))
	}
	assertAdminHeaders(t, rec)
}

func mustAdminToken(t *testing.T) string {
	t.Helper()

	token, err := admindomain.NewTokenManager(admindomain.CookieOptions{}).GenerateToken()
	if err != nil {
		t.Fatalf("expected admin token, got %v", err)
	}
	return token
}

type fakeAdminPanelService struct {
	available     bool
	loginCalled   bool
	loginEmail    string
	loginPassword string
	loginResult   admindomain.LoginResult
	loginErr      error
	resolveCalled bool
	resolveErr    error
	logoutCalled  bool
	logoutErr     error
	dashboard     admindomain.Dashboard
	dashboardErr  error
}

func (s *fakeAdminPanelService) Available() bool {
	return s != nil && s.available
}

func (s *fakeAdminPanelService) Login(_ context.Context, email string, password string) (admindomain.LoginResult, error) {
	s.loginCalled = true
	s.loginEmail = email
	s.loginPassword = password
	if s.loginErr != nil {
		return admindomain.LoginResult{}, s.loginErr
	}
	return s.loginResult, nil
}

func (s *fakeAdminPanelService) ResolveSession(context.Context, *http.Request) (admindomain.Session, error) {
	s.resolveCalled = true
	if s.resolveErr != nil {
		return admindomain.Session{}, s.resolveErr
	}
	return admindomain.Session{AuthUserID: "11111111-1111-1111-1111-111111111111"}, nil
}

func (s *fakeAdminPanelService) Logout(context.Context, *http.Request) error {
	s.logoutCalled = true
	if s.logoutErr != nil {
		return s.logoutErr
	}
	return nil
}

func (s *fakeAdminPanelService) Dashboard(context.Context) (admindomain.Dashboard, error) {
	if s.dashboardErr != nil {
		return admindomain.Dashboard{}, s.dashboardErr
	}
	return s.dashboard, nil
}

func (s *fakeAdminPanelService) WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	admindomain.NewTokenManager(admindomain.CookieOptions{}).WriteCookie(w, token, expiresAt)
}

func (s *fakeAdminPanelService) ClearCookie(w http.ResponseWriter) {
	admindomain.NewTokenManager(admindomain.CookieOptions{}).ClearCookie(w)
}

var errFakeAdminUnavailable = errors.New("admin fake unavailable")
