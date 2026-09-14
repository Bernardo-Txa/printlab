package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
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
		"form-action 'self'",
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

func TestContentSecurityPolicyOmitsSupabaseOriginWhenUnavailable(t *testing.T) {
	policy := contentSecurityPolicy("")

	if !strings.Contains(policy, "img-src 'self' data:") || !strings.Contains(policy, "connect-src 'self'") {
		t.Fatalf("expected CSP to keep self sources, got %q", policy)
	}
	if strings.Contains(policy, "supabase") || strings.Contains(policy, "unsafe-eval") {
		t.Fatalf("expected CSP not to invent external sources, got %q", policy)
	}
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
}
