package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
)

const mfaAdminID = "11111111-1111-1111-1111-111111111111"
const mfaFactorID = "33333333-3333-3333-3333-333333333333"
const mfaOtherID = "44444444-4444-4444-4444-444444444444"

type mfaTestRepo struct {
	admindomain.OrderRepository
	sessions map[string]admindomain.Session
}

func (r *mfaTestRepo) CreateSession(_ context.Context, userID string, hash []byte, expires time.Time) (admindomain.Session, error) {
	now := time.Now()
	s := admindomain.Session{AuthUserID: userID, TokenHash: hash, ExpiresAt: expires, MFAVerifiedAt: &now}
	r.sessions[string(hash)] = s
	return s, nil
}
func (r *mfaTestRepo) ResolveSession(_ context.Context, hash []byte) (admindomain.Session, error) {
	if s, ok := r.sessions[string(hash)]; ok {
		return s, nil
	}
	return admindomain.Session{}, admindomain.ErrSessionNotFound
}
func (r *mfaTestRepo) DeleteSession(_ context.Context, hash []byte) error {
	delete(r.sessions, string(hash))
	return nil
}
func (r *mfaTestRepo) Dashboard(context.Context) (admindomain.Dashboard, error) {
	return admindomain.Dashboard{}, nil
}

type mfaHarness struct {
	handler            http.Handler
	repo               *mfaTestRepo
	factors            []admindomain.MFAFactor
	userID             string
	afterUserID        string
	aal                string
	status             int
	cleanupStatus      int
	getUserStatus      int
	afterGetUserStatus int
	code               string
	selected           string
	deletes            []string
	enrolls            int
	userCalls          int
	verifyCalls        int
	pending            string
	elevated           string
}

func mfaTestJWT(userID, aal string) string {
	data, _ := json.Marshal(map[string]any{"sub": userID, "aal": aal, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
	return "test." + base64.RawURLEncoding.EncodeToString(data) + ".signature"
}

func newMFAHarness(t *testing.T, factors ...admindomain.MFAFactor) *mfaHarness {
	t.Helper()
	h := &mfaHarness{factors: factors, userID: mfaAdminID, afterUserID: mfaAdminID, aal: "aal2", status: 200, cleanupStatus: 200, code: "123456", repo: &mfaTestRepo{sessions: map[string]admindomain.Session{}}}
	h.pending = mfaTestJWT(mfaAdminID, "aal1")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "sb_publishable_test" {
			t.Error("missing publishable key")
		}
		if r.URL.Path != "/auth/v1/token" && r.Header.Get("Authorization") != "Bearer "+h.pending && r.Header.Get("Authorization") != "Bearer "+h.elevated {
			t.Error("missing user token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/auth/v1/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": h.pending, "refresh_token": "never-retain-refresh", "user": map[string]string{"id": h.userID}})
		case r.URL.Path == "/auth/v1/user":
			h.userCalls++
			if r.Header.Get("Authorization") == "Bearer "+h.elevated && h.afterGetUserStatus != 0 {
				w.WriteHeader(h.afterGetUserStatus)
				return
			}
			if h.getUserStatus != 0 {
				w.WriteHeader(h.getUserStatus)
				return
			}
			id := h.userID
			if r.Header.Get("Authorization") == "Bearer "+h.elevated {
				id = h.afterUserID
			}
			_ = json.NewEncoder(w).Encode(admindomain.AuthUser{ID: id, Factors: h.factors})
		case r.Method == http.MethodDelete:
			h.deletes = append(h.deletes, strings.TrimPrefix(r.URL.Path, "/auth/v1/factors/"))
			w.WriteHeader(h.cleanupStatus)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": h.deletes[len(h.deletes)-1]})
		case r.URL.Path == "/auth/v1/factors":
			h.enrolls++
			h.factors = []admindomain.MFAFactor{{ID: mfaFactorID, Type: "totp", Status: "unverified"}}
			_, _ = fmt.Fprintf(w, `{"id":%q,"type":"totp","totp":{"qr_code":"<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>","secret":"MANUAL-SECRET-TEST"}}`, mfaFactorID)
		case strings.HasSuffix(r.URL.Path, "/challenge"):
			h.selected = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/auth/v1/factors/"), "/challenge")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": mfaOtherID})
		case strings.HasSuffix(r.URL.Path, "/verify"):
			h.verifyCalls++
			var input map[string]string
			_ = json.NewDecoder(r.Body).Decode(&input)
			if input["challenge_id"] != mfaOtherID {
				t.Error("missing challenge id")
			}
			if input["code"] != h.code {
				w.WriteHeader(422)
				return
			}
			w.WriteHeader(h.status)
			h.elevated = mfaTestJWT(mfaAdminID, h.aal)
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": h.elevated, "refresh_token": "never-retain-refresh", "user": map[string]string{"id": mfaAdminID}})
		default:
			t.Error("unexpected provider request")
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(provider.Close)
	client, err := admindomain.NewSupabaseAuthClient(admindomain.SupabaseAuthClientConfig{SupabaseURL: provider.URL, PublishableKey: "sb_publishable_test", HTTPClient: provider.Client()})
	if err != nil {
		t.Fatal(err)
	}
	service := admindomain.NewService(client, h.repo, mfaAdminID, admindomain.CookieOptions{Secure: true})
	h.handler = newTestHandlerWithAdmin(t, service, "https://printlab.test")
	return h
}

func (h *mfaHarness) request(method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://printlab.test"+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if method == http.MethodPost {
		req.Header.Set("Origin", "https://printlab.test")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func (h *mfaHarness) pendingCookie() *http.Cookie {
	return &http.Cookie{Name: admindomain.MFAPendingCookieName, Value: h.pending}
}
func verifiedMFAFactor(id string) admindomain.MFAFactor {
	return admindomain.MFAFactor{ID: id, Type: "totp", Status: "verified", FriendlyName: "Autenticador " + id}
}

func TestMFAPasswordStartsOnlyPendingFlow(t *testing.T) {
	for _, hasFactor := range []bool{false, true} {
		t.Run(fmt.Sprint(hasFactor), func(t *testing.T) {
			h := newMFAHarness(t)
			want := "/admin/mfa/setup"
			if hasFactor {
				h.factors = []admindomain.MFAFactor{verifiedMFAFactor(mfaFactorID)}
				want = "/admin/mfa/challenge"
			}
			rec := h.request("POST", "/admin/login", url.Values{"email": {"admin@example.com"}, "password": {"password"}})
			if rec.Code != 303 || rec.Header().Get("Location") != want || len(h.repo.sessions) != 0 {
				t.Fatal("password created session or incorrect MFA destination")
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != admindomain.MFAPendingCookieName || cookies[0].Value != h.pending {
				t.Fatal("expected only pending cookie")
			}
			c := cookies[0]
			if !c.HttpOnly || !c.Secure || c.Domain != "" || c.SameSite != http.SameSiteStrictMode || c.Path != "/admin/mfa" || c.MaxAge != 600 || time.Until(c.Expires) < 590*time.Second || time.Until(c.Expires) > 601*time.Second {
				t.Fatal("invalid pending cookie")
			}
			assertAdminHeaders(t, rec)
		})
	}
	h := newMFAHarness(t)
	h.userID = mfaOtherID
	rec := h.request("POST", "/admin/login", url.Values{"email": {"other@example.com"}, "password": {"password"}})
	if rec.Code != 401 || h.userCalls != 0 || len(h.repo.sessions) != 0 {
		t.Fatal("unauthorized user reached MFA")
	}
}

func TestMFASetupQRAndRetryKeepSecretsPrivate(t *testing.T) {
	h := newMFAHarness(t, admindomain.MFAFactor{ID: mfaOtherID, Type: "totp", Status: "unverified"})
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(old) })
	rec := h.request("GET", "/admin/mfa/setup", nil, h.pendingCookie())
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `src="data:image/svg+xml;base64,`) || !strings.Contains(rec.Body.String(), "MANUAL-SECRET-TEST") {
		t.Fatal("missing enrollment page")
	}
	if len(h.deletes) != 1 || h.deletes[0] != mfaOtherID || h.enrolls != 1 {
		t.Fatal("abandoned factor cleanup incorrect")
	}
	assertAdminHeaders(t, rec)
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "img-src 'self' data:") {
		t.Fatal("QR CSP broken")
	}
	if strings.Contains(rec.Body.String(), h.pending) || strings.Contains(rec.Body.String(), "never-retain-refresh") || strings.Contains(rec.Body.String(), "<svg xmlns") {
		t.Fatal("token or raw SVG in HTML")
	}
	rec = h.request("POST", "/admin/mfa/setup", url.Values{"factor_id": {mfaFactorID}, "code": {"000000"}}, h.pendingCookie())
	if rec.Code != 401 || !strings.Contains(rec.Body.String(), "Código inválido ou expirado.") || strings.Contains(rec.Body.String(), "MANUAL-SECRET-TEST") || h.enrolls != 1 || len(h.repo.sessions) != 0 {
		t.Fatal("invalid setup retry")
	}
	if strings.Contains(logs.String(), "MANUAL-SECRET-TEST") || strings.Contains(logs.String(), h.pending) || strings.Contains(logs.String(), "000000") || strings.Contains(logs.String(), "never-retain-refresh") {
		t.Fatal("sensitive log")
	}
	rec = h.request("POST", "/admin/mfa/setup", url.Values{"factor_id": {mfaFactorID}, "code": {"123456"}}, h.pendingCookie())
	if rec.Code != 303 || rec.Header().Get("Location") != "/admin" || len(h.repo.sessions) != 1 {
		t.Fatal("setup verification did not complete")
	}
}

func TestMFAMultipleFactorsAndFullSessionLifecycle(t *testing.T) {
	h := newMFAHarness(t, verifiedMFAFactor(mfaFactorID), verifiedMFAFactor(mfaOtherID))
	assertAdminRedirectToLogin(t, h.request("GET", "/admin", nil, h.pendingCookie()))
	rec := h.request("GET", "/admin/mfa/challenge", nil, h.pendingCookie())
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<select") || !strings.Contains(rec.Body.String(), mfaFactorID) || !strings.Contains(rec.Body.String(), mfaOtherID) || strings.Contains(rec.Body.String(), "Chave manual") || h.enrolls != 0 {
		t.Fatal("expected factor selector without enrollment")
	}
	assertAdminHeaders(t, rec)
	rec = h.request("POST", "/admin/mfa/challenge", url.Values{"factor_id": {mfaOtherID}, "code": {"123456"}}, h.pendingCookie())
	if rec.Code != 303 || rec.Header().Get("Location") != "/admin" || h.selected != mfaOtherID || len(h.repo.sessions) != 1 {
		t.Fatal("AAL2 did not create session using selected factor")
	}
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == admindomain.CookieName {
			sessionCookie = c
		}
		if c.Name == admindomain.MFAPendingCookieName && c.MaxAge != -1 {
			t.Fatal("pending cookie not cleared")
		}
	}
	if sessionCookie == nil || len(rec.Result().Cookies()) != 2 {
		t.Fatal("missing session or pending cleanup")
	}
	for _, s := range h.repo.sessions {
		if s.MFAVerifiedAt == nil || len(s.TokenHash) != 32 || bytes.Equal(s.TokenHash, []byte(sessionCookie.Value)) {
			t.Fatal("session must be MFA timestamped and hash-only")
		}
	}
	if rec = h.request("GET", "/admin", nil, sessionCookie); rec.Code != 200 {
		t.Fatal("MFA session cannot access admin")
	}
	rec = h.request("POST", "/admin/logout", nil, sessionCookie)
	if len(h.repo.sessions) != 0 || len(rec.Result().Cookies()) != 2 {
		t.Fatal("logout did not clear session and pending cookie")
	}
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge != -1 {
			t.Fatal("logout cookie not expired")
		}
	}
	assertAdminRedirectToLogin(t, h.request("GET", "/admin", nil, sessionCookie))
}

func TestMFARejectsUntrustedFactorsAndProviderFailures(t *testing.T) {
	tests := []struct {
		name           string
		change         func(*mfaHarness)
		factorID, code string
		status         int
	}{
		{"foreign factor", func(h *mfaHarness) {}, mfaOtherID, "123456", 303},
		{"phone", func(h *mfaHarness) { h.factors[0].Type = "phone" }, mfaFactorID, "123456", 303},
		{"unverified", func(h *mfaHarness) { h.factors[0].Status = "unverified" }, mfaFactorID, "123456", 303},
		{"invalid format", func(h *mfaHarness) {}, mfaFactorID, "12345a", 401},
		{"invalid code", func(h *mfaHarness) {}, mfaFactorID, "000000", 401},
		{"rate limit", func(h *mfaHarness) { h.status = 429 }, mfaFactorID, "123456", 429},
		{"unavailable", func(h *mfaHarness) { h.status = 500 }, mfaFactorID, "123456", 503},
		{"not elevated", func(h *mfaHarness) { h.aal = "aal1" }, mfaFactorID, "123456", 303},
		{"wrong user after verify", func(h *mfaHarness) { h.afterUserID = mfaOtherID }, mfaFactorID, "123456", 303},
		{"invalid pending", func(h *mfaHarness) { h.getUserStatus = 401 }, mfaFactorID, "123456", 303},
		{"updated token rejected remotely", func(h *mfaHarness) { h.afterGetUserStatus = 401 }, mfaFactorID, "123456", 303},
		{"updated token validation unavailable", func(h *mfaHarness) { h.afterGetUserStatus = 503 }, mfaFactorID, "123456", 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newMFAHarness(t, verifiedMFAFactor(mfaFactorID))
			tt.change(h)
			rec := h.request("POST", "/admin/mfa/challenge", url.Values{"factor_id": {tt.factorID}, "code": {tt.code}}, h.pendingCookie())
			if rec.Code != tt.status || len(h.repo.sessions) != 0 {
				t.Fatalf("status %d, expected %d; sessions %d", rec.Code, tt.status, len(h.repo.sessions))
			}
			for _, c := range rec.Result().Cookies() {
				if c.Name == admindomain.CookieName && c.MaxAge > 0 {
					t.Fatal("failure created session cookie")
				}
			}
			if tt.status == 303 && rec.Header().Get("Location") != "/admin/login" {
				t.Fatal("terminal failure should require password")
			}
		})
	}
}

func TestMFAOriginsAndMissingCookie(t *testing.T) {
	for _, path := range []string{"/admin/mfa/setup", "/admin/mfa/challenge", "/admin/mfa/cancel"} {
		for _, origin := range []string{"", "https://evil.example", "null", "http://printlab.test"} {
			h := newMFAHarness(t)
			req := httptest.NewRequest("POST", "https://printlab.test"+path, strings.NewReader("code=123456"))
			if origin != "" {
				req.Header.Set("Origin", origin)
			}
			req.AddCookie(h.pendingCookie())
			rec := httptest.NewRecorder()
			h.handler.ServeHTTP(rec, req)
			if rec.Code != 403 || h.userCalls != 0 {
				t.Fatal("MFA mutation source accepted")
			}
		}
	}
	h := newMFAHarness(t)
	assertAdminRedirectToLogin(t, h.request("GET", "/admin/mfa/setup", nil))
	assertAdminRedirectToLogin(t, h.request("GET", "/admin/mfa/challenge", nil))
	rec := h.request("POST", "/admin/mfa/cancel", nil, h.pendingCookie())
	if rec.Code != 303 || rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("cancel did not clear pending cookie")
	}
}

func TestMFASetupPreservesVerifiedFactorsAndStopsOnCleanupFailure(t *testing.T) {
	h := newMFAHarness(t, verifiedMFAFactor(mfaFactorID))
	rec := h.request("GET", "/admin/mfa/setup", nil, h.pendingCookie())
	if rec.Code != 303 || rec.Header().Get("Location") != "/admin/mfa/challenge" || h.enrolls != 0 || len(h.deletes) != 0 {
		t.Fatal("verified factor modified by setup")
	}
	h = newMFAHarness(t, admindomain.MFAFactor{ID: mfaFactorID, Type: "totp", Status: "unverified"})
	h.cleanupStatus = 500
	rec = h.request("GET", "/admin/mfa/setup", nil, h.pendingCookie())
	if rec.Code != 503 || h.enrolls != 0 {
		t.Fatal("cleanup failure must stop enrollment")
	}
}

func (s *fakeAdminPanelService) MFAPage(context.Context, *http.Request, bool) (admindomain.MFAPage, error) {
	return admindomain.MFAPage{}, admindomain.ErrUnauthenticated
}
func (s *fakeAdminPanelService) CompleteMFA(context.Context, *http.Request, bool, string, string) (admindomain.LoginResult, admindomain.MFAPage, error) {
	return admindomain.LoginResult{}, admindomain.MFAPage{}, admindomain.ErrUnauthenticated
}
func (s *fakeAdminPanelService) WritePendingCookie(w http.ResponseWriter, token string, expires time.Time) {
	service := admindomain.NewService(nil, nil, mfaAdminID, admindomain.CookieOptions{})
	service.WritePendingCookie(w, token, expires)
}
func (s *fakeAdminPanelService) ClearPendingCookie(w http.ResponseWriter) {
	service := admindomain.NewService(nil, nil, mfaAdminID, admindomain.CookieOptions{})
	service.ClearPendingCookie(w)
}
