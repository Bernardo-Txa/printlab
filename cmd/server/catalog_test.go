package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

func TestCatalogEmptyReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithCatalog(t, &fakeCatalogService{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Os primeiros experimentos estao quase prontos.") {
		t.Fatal("expected empty catalog state")
	}
}

func TestCatalogWithProductsReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{
		catalog: products.Catalog{
			Products: []products.Product{
				{Name: "Produto Real", Slug: "produto-real", PriceBRL: "R$ 29,90", PriceFrom: true},
			},
		},
	}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Produto Real") || !strings.Contains(body, "R$ 29,90") || !strings.Contains(body, "A partir de") {
		t.Fatal("expected rendered product card")
	}
}

func TestCatalogForwardsCategoryFilter(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos?categoria=articulados", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if service.lastFilter.CategorySlug != "articulados" {
		t.Fatalf("expected category filter to be forwarded, got %q", service.lastFilter.CategorySlug)
	}
}

func TestProductFoundReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{
		detail: products.ProductDetail{
			Product: products.Product{
				Name:             "Produto Real",
				Slug:             "produto-real",
				ShortDescription: "Objeto impresso pela PrintLab.",
				Description:      "Descricao completa do produto.",
			},
			DisplayPriceBRL:   "R$ 100,00",
			DisplayPriceLabel: "Preco-base",
			CanonicalPath:     "/produtos/produto-real",
		},
	}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Produto Real") || !strings.Contains(body, "R$ 100,00") {
		t.Fatal("expected rendered product detail")
	}

	if !strings.Contains(body, "Imagem em preparo") {
		t.Fatal("expected placeholder image fallback")
	}

	if !strings.Contains(body, `action="/carrinho/adicionar"`) || !strings.Contains(body, `name="product_slug"`) {
		t.Fatal("expected real add-to-cart form")
	}

	for _, forbiddenField := range []string{`name="unit_price"`, `name="subtotal"`, `name="total"`, `name="product_name"`} {
		if strings.Contains(body, forbiddenField) {
			t.Fatalf("expected product form not to include authoritative field %s", forbiddenField)
		}
	}
}

func TestProductWithDefaultVariantReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{
		detail: products.ProductDetail{
			Product: products.Product{Name: "Produto Real", Slug: "produto-real"},
			Variants: []products.ProductVariant{
				{Name: "Mini", Slug: "mini", EffectivePriceBRL: "R$ 29,90"},
				{Name: "Padrao", Slug: "padrao", EffectivePriceBRL: "R$ 39,90", IsDefault: true},
			},
			SelectedVariant:   &products.ProductVariant{Name: "Padrao", Slug: "padrao", EffectivePriceBRL: "R$ 39,90", IsDefault: true},
			DisplayPriceBRL:   "R$ 39,90",
			DisplayPriceLabel: "Preco-base",
			CanonicalPath:     "/produtos/produto-real",
		},
	}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Padrao") || !strings.Contains(body, `aria-current="true"`) {
		t.Fatal("expected default variant to be rendered as selected")
	}
}

func TestProductWithVariantQueryReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real?variante=grande", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{
		detail: products.ProductDetail{
			Product: products.Product{Name: "Produto Real", Slug: "produto-real"},
			Variants: []products.ProductVariant{
				{Name: "Mini", Slug: "mini", EffectivePriceBRL: "R$ 29,90"},
				{Name: "Grande", Slug: "grande", EffectivePriceBRL: "R$ 59,90"},
			},
			SelectedVariant:   &products.ProductVariant{Name: "Grande", Slug: "grande", EffectivePriceBRL: "R$ 59,90"},
			DisplayPriceBRL:   "R$ 59,90",
			DisplayPriceLabel: "Preco da variante",
			CanonicalPath:     "/produtos/produto-real",
		},
	}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if service.lastVariantSlug != "grande" {
		t.Fatalf("expected variant slug to be forwarded, got %q", service.lastVariantSlug)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Grande") || !strings.Contains(body, "R$ 59,90") {
		t.Fatal("expected selected variant to be rendered")
	}

	if !strings.Contains(body, `rel="canonical" href="/produtos/produto-real"`) {
		t.Fatal("expected product canonical URL to ignore variant query")
	}
}

func TestProductVariantNotFoundReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real?variante=grande", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{productErr: products.ErrNotFound}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestInvalidVariantSlugDoesNotCallService(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real?variante=Grande", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if service.productCalls != 0 {
		t.Fatal("expected invalid variant slug not to call service")
	}
}

func TestProductNotFoundReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/nao-existe", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{productErr: products.ErrNotFound}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestInvalidProductSlugDoesNotCallService(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/Produto", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if service.productCalls != 0 {
		t.Fatal("expected invalid slug not to call service")
	}
}

func TestCatalogUnavailableDoesNotLeakDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{catalogErr: errors.New("postgres password=secret")}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	body := strings.ToLower(rec.Body.String())
	for _, term := range []string{"postgres", "password", "secret"} {
		if strings.Contains(body, term) {
			t.Fatalf("expected response not to leak %q", term)
		}
	}
}

func TestProductUnavailableDoesNotLeakDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos/produto-real", nil)
	rec := httptest.NewRecorder()
	service := &fakeCatalogService{productErr: errors.New("postgres password=secret")}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	body := strings.ToLower(rec.Body.String())
	for _, term := range []string{"postgres", "password", "secret"} {
		if strings.Contains(body, term) {
			t.Fatalf("expected response not to leak %q", term)
		}
	}
}

func TestCatalogWithoutDatabaseReturns503(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/produtos", nil)
	rec := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

type fakeCatalogService struct {
	catalog products.Catalog
	detail  products.ProductDetail

	catalogErr error
	productErr error

	lastFilter      products.ListFilter
	lastSlug        string
	lastVariantSlug string
	productCalls    int
}

func (s *fakeCatalogService) Catalog(_ context.Context, filter products.ListFilter) (products.Catalog, error) {
	s.lastFilter = filter
	if s.catalogErr != nil {
		return products.Catalog{}, s.catalogErr
	}

	return s.catalog, nil
}

func (s *fakeCatalogService) Product(_ context.Context, slug string, variantSlug string) (products.ProductDetail, error) {
	s.productCalls++
	s.lastSlug = slug
	s.lastVariantSlug = variantSlug
	if s.productErr != nil {
		return products.ProductDetail{}, s.productErr
	}

	return s.detail, nil
}
