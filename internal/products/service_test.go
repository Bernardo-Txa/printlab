package products

import (
	"context"
	"errors"
	"testing"
)

func TestCatalogRejectsInvalidCategorySlugBeforeRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Catalog(context.Background(), ListFilter{CategorySlug: "Categoria"})
	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}

	if repository.listCategoriesCalls != 0 || repository.listProductsCalls != 0 {
		t.Fatal("expected invalid slug not to call repository")
	}
}

func TestCatalogFormatsProductPricesAndSelectedCategory(t *testing.T) {
	repository := &fakeRepository{
		categories: []Category{
			{ID: "cat-1", Name: "Articulados", Slug: "articulados"},
		},
		products: []Product{
			{ID: "prod-1", Name: "Produto Real", Slug: "produto-real", PriceCents: 3990},
		},
	}
	service := NewService(repository)

	catalog, err := service.Catalog(context.Background(), ListFilter{CategorySlug: "articulados"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.lastFilter.CategorySlug != "articulados" {
		t.Fatalf("expected category filter to be forwarded, got %q", repository.lastFilter.CategorySlug)
	}

	if catalog.SelectedCategory == nil || catalog.SelectedCategory.Name != "Articulados" {
		t.Fatal("expected selected category to be resolved")
	}

	if got := catalog.Products[0].PriceBRL; got != "R$ 39,90" {
		t.Fatalf("expected formatted price, got %q", got)
	}
}

func TestProductRejectsInvalidSlugBeforeRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Product(context.Background(), "../produto")
	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}

	if repository.productCalls != 0 {
		t.Fatal("expected invalid slug not to call repository")
	}
}

func TestProductReturnsNotFound(t *testing.T) {
	service := NewService(&fakeRepository{productErr: ErrNotFound})

	_, err := service.Product(context.Background(), "produto-real")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProductHidesRepositoryErrors(t *testing.T) {
	service := NewService(&fakeRepository{productErr: errors.New("postgres password=secret")})

	_, err := service.Product(context.Background(), "produto-real")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

type fakeRepository struct {
	categories []Category
	products   []Product
	product    Product

	listCategoriesErr error
	listProductsErr   error
	productErr        error

	listCategoriesCalls int
	listProductsCalls   int
	productCalls        int
	lastFilter          ListFilter
	lastSlug            string
}

func (r *fakeRepository) ListActiveCategories(_ context.Context) ([]Category, error) {
	r.listCategoriesCalls++
	if r.listCategoriesErr != nil {
		return nil, r.listCategoriesErr
	}

	return r.categories, nil
}

func (r *fakeRepository) ListActiveProducts(_ context.Context, filter ListFilter) ([]Product, error) {
	r.listProductsCalls++
	r.lastFilter = filter
	if r.listProductsErr != nil {
		return nil, r.listProductsErr
	}

	return r.products, nil
}

func (r *fakeRepository) GetActiveProductBySlug(_ context.Context, slug string) (Product, error) {
	r.productCalls++
	r.lastSlug = slug
	if r.productErr != nil {
		return Product{}, r.productErr
	}

	return r.product, nil
}
