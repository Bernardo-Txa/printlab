package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected content type %q, got %q", "text/html; charset=utf-8", got)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "PrintLab") {
		t.Fatal("expected response to identify PrintLab")
	}

	if !strings.Contains(body, "Pular para o conteúdo") {
		t.Fatal("expected response to include the skip link")
	}

	if !strings.Contains(body, "/static/images/branding/logo-printlab-primary.png") {
		t.Fatal("expected response to reference the PrintLab logo")
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Body.String(); got != "ok\n" {
		t.Fatalf("expected body %q, got %q", "ok\n", got)
	}
}

func TestStaticCSSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/css/app.css", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("expected CSS content type, got %q", got)
	}
}

func TestStaticBrandLogoHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/images/branding/logo-printlab-primary.png", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
		t.Fatalf("expected PNG content type, got %q", got)
	}
}

func TestStaticDirectoryListingIsNotServed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/css/", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestMissingStaticFile(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/css/missing.css", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUnknownRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nao-existe", nil)
	rec := httptest.NewRecorder()

	newHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
