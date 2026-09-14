package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
)

func TestGlobalSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	assertGlobalSecurityHeaders(t, rec)
	if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("expected global referrer policy, got %q", rec.Header().Get("Referrer-Policy"))
	}
}

func TestCanonicalHostRedirectsGETToConfiguredSiteURL(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "https://www.printlab3d.com.br", "")
	req := httptest.NewRequest(http.MethodGet, "https://printlab3d.com.br/produtos/produto-real?a=1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusPermanentRedirect, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://www.printlab3d.com.br/produtos/produto-real?a=1" {
		t.Fatalf("expected canonical redirect preserving path and query, got %q", got)
	}
}

func TestCanonicalHostDoesNotRedirectConfiguredHost(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "https://www.printlab3d.com.br", "")
	req := httptest.NewRequest(http.MethodGet, "https://www.printlab3d.com.br/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected canonical host to reach handler with status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCanonicalHostAllowsConfiguredLocalhost(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "http://localhost:8080", "")
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected localhost canonical host to reach handler with status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCanonicalHostRedirectDestinationIgnoresRequestHost(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "https://www.printlab3d.com.br", "")
	req := httptest.NewRequest(http.MethodGet, "https://malicious.example/checkout/revisao?step=3", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusPermanentRedirect, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://www.printlab3d.com.br/checkout/revisao?step=3" {
		t.Fatalf("expected redirect destination to use only SITE_URL origin, got %q", got)
	}
	if strings.Contains(rec.Header().Get("Location"), "malicious.example") {
		t.Fatalf("expected redirect destination host not to use request host, got %q", rec.Header().Get("Location"))
	}
}

func TestGlobalSecurityHeadersIncludeConfiguredSupabaseOrigin(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "https://example.supabase.co/path")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' data: https://example.supabase.co") ||
		!strings.Contains(csp, "connect-src 'self' https://example.supabase.co") {
		t.Fatalf("expected CSP to include configured Supabase origin, got %q", csp)
	}
}

func TestGlobalSecurityHeadersPreserveAdminReferrerPolicy(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, &fakeAdminPanelService{available: true}, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertGlobalSecurityHeaders(t, rec)
	if rec.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatalf("expected admin referrer policy to be preserved, got %q", rec.Header().Get("Referrer-Policy"))
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected admin cache control to be preserved, got %q", rec.Header().Get("Cache-Control"))
	}
	if robots := rec.Header().Get("X-Robots-Tag"); !strings.Contains(robots, "noindex") {
		t.Fatalf("expected admin robots header to be preserved, got %q", robots)
	}
}

func TestGlobalSecurityHeadersPreserveTrackingReferrerPolicy(t *testing.T) {
	handler := newTestHandlerWithOrders(t, &fakeOrderReviewService{trackingPage: trackingPageFixture()}, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/acompanhar/"+trackingID, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertGlobalSecurityHeaders(t, rec)
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected tracking referrer policy to be preserved, got %q", rec.Header().Get("Referrer-Policy"))
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected tracking cache control to be preserved, got %q", rec.Header().Get("Cache-Control"))
	}
	if robots := rec.Header().Get("X-Robots-Tag"); !strings.Contains(robots, "noindex") {
		t.Fatalf("expected tracking robots header to be preserved, got %q", robots)
	}
}

func TestContentSecurityPolicyUsesSupabaseOriginOnly(t *testing.T) {
	policy := contentSecurityPolicy("https://example.supabase.co/project/path?query=1")

	for _, required := range []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https://example.supabase.co",
		"connect-src 'self' https://example.supabase.co",
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'none'",
		"form-action 'self' https://checkout.infinitepay.io https://checkout.infinitepay.com.br",
	} {
		if !strings.Contains(policy, required) {
			t.Fatalf("expected CSP to contain %q, got %q", required, policy)
		}
	}
	for _, forbidden := range []string{"unsafe-eval", "/project", "?query=1"} {
		if strings.Contains(policy, forbidden) {
			t.Fatalf("expected CSP not to contain %q, got %q", forbidden, policy)
		}
	}
}

func TestContentSecurityPolicyFormActionUsesExactCheckoutAllowlist(t *testing.T) {
	policy := contentSecurityPolicy("")
	got, ok := cspDirectiveSources(policy, "form-action")
	if !ok {
		t.Fatalf("expected form-action directive in CSP, got %q", policy)
	}

	want := append([]string{"'self'"}, paymentsdomain.CheckoutAllowedOrigins()...)
	if !slices.Equal(got, want) {
		t.Fatalf("expected form-action sources %v, got %v", want, got)
	}

	for _, forbidden := range []string{
		"https:",
		"*",
		"*.infinitepay.io",
		"https://*.infinitepay.io",
		"https://api.checkout.infinitepay.io",
	} {
		if slices.Contains(got, forbidden) {
			t.Fatalf("expected form-action not to allow %q, got %v", forbidden, got)
		}
	}

	connectSrc, ok := cspDirectiveSources(policy, "connect-src")
	if !ok {
		t.Fatalf("expected connect-src directive in CSP, got %q", policy)
	}
	for _, origin := range paymentsdomain.CheckoutAllowedOrigins() {
		if slices.Contains(connectSrc, origin) {
			t.Fatalf("expected checkout origin %q not to be added to connect-src, got %v", origin, connectSrc)
		}
	}
}

func TestContentSecurityPolicyOmitsSupabaseOriginWhenUnavailable(t *testing.T) {
	policy := contentSecurityPolicy("")

	if !strings.Contains(policy, "img-src 'self' data:") || !strings.Contains(policy, "connect-src 'self'") {
		t.Fatalf("expected CSP to keep self sources, got %q", policy)
	}
	if strings.Contains(policy, "supabase") || strings.Contains(policy, "unsafe-eval") {
		t.Fatalf("expected CSP not to invent external sources, got %q", policy)
	}
}

func cspDirectiveSources(policy string, name string) ([]string, bool) {
	for _, directive := range strings.Split(policy, ";") {
		fields := strings.Fields(strings.TrimSpace(directive))
		if len(fields) > 0 && fields[0] == name {
			return fields[1:], true
		}
	}

	return nil, false
}

func TestGlobalRequestBodyLimitRejectsKnownOversizedBody(t *testing.T) {
	service := &fakeCartService{}
	body := strings.NewReader(strings.Repeat("a", globalMaxRequestBodyBytes+1))
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/carrinho/adicionar", body)
	req.ContentLength = globalMaxRequestBodyBytes + 1
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
	if service.addCalls != 0 {
		t.Fatal("expected oversized body not to call cart service")
	}
}

func TestGlobalRequestBodyLimitRejectsUnknownOversizedBody(t *testing.T) {
	service := &fakeCartService{}
	formBody := "quantity=1&product_slug=" + strings.Repeat("a", globalMaxRequestBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/carrinho/adicionar", strings.NewReader(formBody))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
	if service.addCalls != 0 {
		t.Fatal("expected oversized body not to call cart service")
	}
}

func TestAdminImageJSONLimitStillAppliesBelowGlobalLimit(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(
		http.MethodPost,
		"https://printlab.test/admin/produtos/11111111-1111-1111-1111-111111111111/imagens/upload-url",
		strings.NewReader(strings.Repeat(" ", adminMaxJSONBodyBytes+1)),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://printlab.test")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected admin JSON limit to return status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
	if service.imageAuthCalled {
		t.Fatal("expected oversized admin image JSON body not to call service")
	}
}

func assertGlobalSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected DENY frame header, got %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Permissions-Policy") != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("expected permissions policy, got %q", rec.Header().Get("Permissions-Policy"))
	}
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("expected CSP baseline, got %q", csp)
	}
	if strings.Contains(csp, "unsafe-eval") {
		t.Fatalf("expected CSP without unsafe-eval, got %q", csp)
	}
	if !strings.Contains(csp, "form-action 'self'") {
		t.Fatalf("expected CSP to keep form-action self, got %q", csp)
	}
}
