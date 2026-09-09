package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
)

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

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

	if !strings.Contains(body, "Imprimimos") || !strings.Contains(body, "Por que Lab?") {
		t.Fatal("expected response to include the brand experience sections")
	}
}

func TestHomeCopyDoesNotExposeTechnicalImplementation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	body := strings.ToLower(rec.Body.String())
	for _, term := range []string{"backend", "server-side", "banco", "go:embed"} {
		if strings.Contains(body, term) {
			t.Fatalf("expected homepage copy not to expose technical term %q", term)
		}
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("expected content type %q, got %q", "text/plain; charset=utf-8", got)
	}

	if got := rec.Body.String(); got != "ok\n" {
		t.Fatalf("expected body %q, got %q", "ok\n", got)
	}
}

func TestReadyWithoutDatabaseURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	if strings.Contains(rec.Body.String(), "postgres") {
		t.Fatal("expected ready response not to expose database details")
	}
}

func TestStaticCSSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/css/app.css", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

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

	newTestHandler(t).ServeHTTP(rec, req)

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

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestMissingStaticFile(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/css/missing.css", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUnknownRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nao-existe", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandler(db)
}
