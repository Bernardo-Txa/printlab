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
				{Name: "Produto Real", Slug: "produto-real", PriceBRL: "R$ 39,90"},
			},
		},
	}

	newTestHandlerWithCatalog(t, service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Produto Real") || !strings.Contains(body, "R$ 39,90") {
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
		product: products.Product{
			Name:             "Produto Real",
			Slug:             "produto-real",
			ShortDescription: "Objeto impresso pela PrintLab.",
			Description:      "Descricao completa do produto.",
			PriceBRL:         "R$ 100,00",
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
	product products.Product

	catalogErr error
	productErr error

	lastFilter   products.ListFilter
	lastSlug     string
	productCalls int
}

func (s *fakeCatalogService) Catalog(_ context.Context, filter products.ListFilter) (products.Catalog, error) {
	s.lastFilter = filter
	if s.catalogErr != nil {
		return products.Catalog{}, s.catalogErr
	}

	return s.catalog, nil
}

func (s *fakeCatalogService) Product(_ context.Context, slug string) (products.Product, error) {
	s.productCalls++
	s.lastSlug = slug
	if s.productErr != nil {
		return products.Product{}, s.productErr
	}

	return s.product, nil
}
