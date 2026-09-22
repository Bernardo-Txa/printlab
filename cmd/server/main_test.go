package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
	"github.com/Bernardo-Txa/printlab/web/components"
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

	for _, asset := range []string{
		"/static/images/branding/logo-printlab-hero-v1.webp",
		"/static/images/branding/logo-printlab-small-v1.webp",
		"/static/images/branding/favicon-printlab-v1.png",
	} {
		if !strings.Contains(body, asset) {
			t.Fatalf("expected response to reference optimized branding asset %q", asset)
		}
	}
	for _, obsoleteARIA := range []string{
		`aria-label="Identidade visual da PrintLab"`,
		`aria-label="Espaco visual reservado para catalogo futuro"`,
		`aria-label="Falar com a PrintLab"`,
	} {
		if strings.Contains(body, obsoleteARIA) {
			t.Fatalf("expected homepage not to use redundant ARIA label %q", obsoleteARIA)
		}
	}

	if !strings.Contains(body, "Imprimimos") || !strings.Contains(body, "Por que Lab?") {
		t.Fatal("expected response to include the brand experience sections")
	}

	if !strings.Contains(body, components.WhatsAppGeneralURL()) || !strings.Contains(body, "Falar com a PrintLab no WhatsApp") {
		t.Fatal("expected homepage contact CTA to point to official WhatsApp")
	}
	if strings.Contains(body, "O canal oficial de atendimento será definido em breve") {
		t.Fatal("expected homepage not to contain obsolete contact copy")
	}
	for _, expected := range []string{"PrintLab | Impressão 3D", "Laboratório de impressão 3D", "Precisão", "objetos físicos", "linguagem própria", "fabricação digital", "técnica", "imaginação", "propósito", "espaço", "apresentação", "frasco de laboratório"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected homepage copy to contain %q", expected)
		}
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
	if got := rec.Header().Get("X-Robots-Tag"); got != "noindex, nofollow, noarchive" {
		t.Fatalf("expected health endpoint to be noindex, got %q", got)
	}
}

func TestHTTPServerTimeouts(t *testing.T) {
	server := newHTTPServer(":0", http.NotFoundHandler())

	if server.ReadHeaderTimeout != serverReadHeaderTimeout ||
		server.ReadTimeout != serverReadTimeout ||
		server.WriteTimeout != serverWriteTimeout ||
		server.IdleTimeout != serverIdleTimeout {
		t.Fatalf("expected conservative server timeouts, got %#v", server)
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
	if got := rec.Header().Get("X-Robots-Tag"); got != "noindex, nofollow, noarchive" {
		t.Fatalf("expected ready endpoint to be noindex, got %q", got)
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
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("expected conservative static cache policy, got %q", got)
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

func TestVersionedStaticAssetUsesLongImmutableCache(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/images/branding/logo-printlab-small-v1.webp", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("expected immutable cache policy for versioned asset, got %q", got)
	}
}

func TestStaticCheckoutJSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/js/checkout.js", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") && !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("expected JavaScript content type, got %q", got)
	}
}

func TestStaticAdminImagesJSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/js/admin-images.js", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") && !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("expected JavaScript content type, got %q", got)
	}
}

func TestStaticMotionJSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/js/motion.js", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") && !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("expected JavaScript content type, got %q", got)
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

	return newHandlerWithCatalog(db, nil)
}

func newTestHandlerWithCatalog(t *testing.T, service catalogService) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithCatalog(db, service)
}
