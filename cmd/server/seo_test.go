package main

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const seoTestSiteURL = "https://www.printlab.test"

func TestPublicPagesUseConfiguredAbsoluteCanonicalAndMetadata(t *testing.T) {
	service := &fakeCatalogService{
		catalog: products.Catalog{},
		detail: products.ProductDetail{
			Product: products.Product{
				Name:             "Produto Real",
				Slug:             "produto-real",
				ShortDescription: "Descricao do produto.",
			},
			CanonicalPath: "/produtos/produto-real",
		},
	}
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, service, nil, nil, nil, nil, nil, nil, nil, nil, seoTestSiteURL, "")

	for _, test := range []struct {
		path      string
		canonical string
	}{
		{"/", seoTestSiteURL + "/"},
		{"/produtos?categoria=teste", seoTestSiteURL + "/produtos"},
		{"/produtos/produto-real?variante=grande", seoTestSiteURL + "/produtos/produto-real"},
	} {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, seoTestSiteURL+test.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `rel="canonical" href="`+test.canonical+`"`) {
				t.Fatalf("expected configured absolute canonical %q, got %s", test.canonical, body)
			}
			for _, metadata := range []string{"og:title", "og:description", "og:type", "og:url", `name="twitter:card"`} {
				if !strings.Contains(body, metadata) {
					t.Fatalf("expected metadata %q", metadata)
				}
			}
		})
	}
}

func TestTransactionalPagesSetNoIndexHeaderAndMeta(t *testing.T) {
	handler := newHandlerWithServicesAndOrdersAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, seoTestSiteURL, "")

	for _, path := range []string{"/carrinho", "/checkout/dados", "/pedido/teste", "/acompanhar/teste", "/pagamento/retorno", "/admin/login"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, seoTestSiteURL+path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("X-Robots-Tag"); got != "noindex, nofollow, noarchive" {
				t.Fatalf("expected noindex header, got %q", got)
			}
		})
	}

	for _, path := range []string{"/carrinho", "/pagamento/retorno", "/admin/login"} {
		t.Run(path+" HTML", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, seoTestSiteURL+path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if body := rec.Body.String(); !strings.Contains(body, `name="robots" content="noindex, nofollow, noarchive"`) {
				t.Fatalf("expected noindex meta tag, got %s", body)
			}
		})
	}
}

func TestRobotsHandlerPublishesSitemapAndPrivateHints(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	robotsHandler(seoTestSiteURL).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, required := range []string{"User-agent: *", "Allow: /", "Disallow: /admin", "Disallow: /checkout", "Disallow: /carrinho", "Disallow: /pedido", "Disallow: /acompanhar", "Disallow: /pagamento", "Sitemap: " + seoTestSiteURL + "/sitemap.xml"} {
		if !strings.Contains(body, required) {
			t.Fatalf("expected robots.txt to contain %q", required)
		}
	}
	for _, obsolete := range []string{"Disallow: /admin/", "Disallow: /checkout/", "Disallow: /pedido/", "Disallow: /acompanhar/", "Disallow: /pagamento/"} {
		if strings.Contains(body, obsolete) {
			t.Fatalf("expected robots.txt not to contain %q", obsolete)
		}
	}
}

func TestSitemapXMLSerializesLocationSafely(t *testing.T) {
	body, err := xml.Marshal(sitemap{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  []sitemapURL{{Location: "https://www.printlab.test/produtos/item?cor=azul&material=pla"}},
	})
	if err != nil {
		t.Fatalf("expected sitemap serialization to succeed, got %v", err)
	}

	if got := string(body); !strings.Contains(got, "cor=azul&amp;material=pla") {
		t.Fatalf("expected XML location escaping, got %s", got)
	}
}

func TestSitemapContainsOnlyPublicCanonicalURLs(t *testing.T) {
	service := &fakeCatalogService{catalog: products.Catalog{Products: []products.Product{
		{Slug: "produto-ativo"},
		{Slug: "slug-invalido!"},
	}}}
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()
	sitemapHandler(service, seoTestSiteURL).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, required := range []string{seoTestSiteURL + "/", seoTestSiteURL + "/produtos", seoTestSiteURL + "/produtos/produto-ativo"} {
		if !strings.Contains(body, required) {
			t.Fatalf("expected sitemap to contain %q", required)
		}
	}
	for _, forbidden := range []string{"slug-invalido", "/checkout/", "/pedido/"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected sitemap not to contain %q", forbidden)
		}
	}
}

func TestSitemapFailsSafelyWithoutCatalogOrValidSiteURL(t *testing.T) {
	for _, test := range []struct {
		name    string
		service catalogService
		siteURL string
	}{
		{name: "catalog unavailable", siteURL: seoTestSiteURL},
		{name: "invalid site URL", service: &fakeCatalogService{}, siteURL: "//untrusted.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
			rec := httptest.NewRecorder()
			sitemapHandler(test.service, test.siteURL).ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
			}
		})
	}
}
