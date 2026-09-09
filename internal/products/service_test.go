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

func TestCatalogFormatsMinimumVariantPrice(t *testing.T) {
	repository := &fakeRepository{
		products: []Product{
			{
				ID:                "prod-1",
				Name:              "Produto Real",
				Slug:              "produto-real",
				PriceCents:        3990,
				DisplayPriceCents: 2990,
				HasDisplayPrice:   true,
				PriceFrom:         true,
			},
		},
	}
	service := NewService(repository)

	catalog, err := service.Catalog(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	product := catalog.Products[0]
	if product.PriceBRL != "R$ 29,90" {
		t.Fatalf("expected minimum effective price, got %q", product.PriceBRL)
	}

	if !product.PriceFrom {
		t.Fatal("expected product card to mark price as starting price")
	}
}

func TestProductRejectsInvalidSlugBeforeRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Product(context.Background(), "../produto", "")
	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}

	if repository.productCalls != 0 {
		t.Fatal("expected invalid slug not to call repository")
	}
}

func TestProductRejectsInvalidVariantSlugBeforeRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Product(context.Background(), "produto-real", "Grande")
	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("expected ErrInvalidSlug, got %v", err)
	}

	if repository.productCalls != 0 {
		t.Fatal("expected invalid variant slug not to call repository")
	}
}

func TestProductReturnsNotFound(t *testing.T) {
	service := NewService(&fakeRepository{productErr: ErrNotFound})

	_, err := service.Product(context.Background(), "produto-real", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProductHidesRepositoryErrors(t *testing.T) {
	service := NewService(&fakeRepository{productErr: errors.New("postgres password=secret")})

	_, err := service.Product(context.Background(), "produto-real", "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestProductSelectsDefaultVariant(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			activeVariant("variant-mini", "Mini", false),
			activeVariant("variant-padrao", "Padrao", true),
		}),
	})

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant == nil || detail.SelectedVariant.Slug != "variant-padrao" {
		t.Fatalf("expected default variant to be selected, got %#v", detail.SelectedVariant)
	}
}

func TestProductSelectsFirstVariantWhenNoDefaultExists(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			activeVariant("variant-mini", "Mini", false),
			activeVariant("variant-grande", "Grande", false),
		}),
	})

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant == nil || detail.SelectedVariant.Slug != "variant-mini" {
		t.Fatalf("expected first active variant to be selected, got %#v", detail.SelectedVariant)
	}
}

func TestProductSelectsExplicitVariant(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			activeVariant("variant-mini", "Mini", true),
			activeVariant("variant-grande", "Grande", false),
		}),
	})

	detail, err := service.Product(context.Background(), "produto-real", "variant-grande")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant == nil || detail.SelectedVariant.Slug != "variant-grande" {
		t.Fatalf("expected explicit variant to be selected, got %#v", detail.SelectedVariant)
	}
}

func TestProductReturnsNotFoundForMissingVariant(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			activeVariant("variant-mini", "Mini", true),
		}),
	})

	_, err := service.Product(context.Background(), "produto-real", "variant-grande")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProductIgnoresInactiveVariant(t *testing.T) {
	inactive := activeVariant("variant-grande", "Grande", false)
	inactive.IsActive = false
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{inactive}),
	})

	_, err := service.Product(context.Background(), "produto-real", "variant-grande")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProductIgnoresVariantFromAnotherProduct(t *testing.T) {
	otherProductVariant := activeVariant("variant-grande", "Grande", false)
	otherProductVariant.ProductID = "prod-2"
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{otherProductVariant}),
	})

	_, err := service.Product(context.Background(), "produto-real", "variant-grande")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProductWithoutVariantsUsesBasePrice(t *testing.T) {
	service := NewService(&fakeRepository{detail: productDetailFixture(nil)})

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant != nil {
		t.Fatalf("expected no selected variant, got %#v", detail.SelectedVariant)
	}

	if detail.DisplayPriceBRL != "R$ 39,90" {
		t.Fatalf("expected base price, got %q", detail.DisplayPriceBRL)
	}
}

func TestProductPreparesVariantMaterialsAndColors(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			{
				ID:        "variant-1",
				ProductID: "prod-1",
				Name:      "Padrao",
				Slug:      "padrao",
				IsActive:  true,
				IsDefault: true,
				Filaments: []VariantFilament{
					{
						VariantID:         "variant-1",
						Material:          Material{ID: "mat-pla", Name: "PLA", Slug: "pla"},
						Color:             Color{ID: "cor-azul", Name: "Azul", Slug: "azul", HexColor: "#0066FF"},
						EstimatedWeightMg: 32000,
					},
					{
						VariantID:         "variant-1",
						Material:          Material{ID: "mat-petg", Name: "PETG", Slug: "petg"},
						Color:             Color{ID: "cor-branco", Name: "Branco", Slug: "branco", HexColor: "#FFFFFF"},
						EstimatedWeightMg: 7000,
					},
					{
						VariantID:         "variant-1",
						Material:          Material{ID: "mat-pla", Name: "PLA", Slug: "pla"},
						Color:             Color{ID: "cor-rosa", Name: "Rosa", Slug: "rosa", HexColor: "#FF4FA3"},
						EstimatedWeightMg: 3000,
					},
				},
			},
		}),
	})

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant == nil {
		t.Fatal("expected selected variant")
	}

	if detail.SelectedVariant.TotalWeightMg != 42000 || detail.SelectedVariant.TotalWeightGrams != "42 g" {
		t.Fatalf("expected total weight to be prepared, got %d %q", detail.SelectedVariant.TotalWeightMg, detail.SelectedVariant.TotalWeightGrams)
	}

	if detail.SelectedVariant.MaterialNames != "PLA, PETG" {
		t.Fatalf("expected material names, got %q", detail.SelectedVariant.MaterialNames)
	}

	if detail.SelectedVariant.ColorNames != "Azul, Branco, Rosa" {
		t.Fatalf("expected color names, got %q", detail.SelectedVariant.ColorNames)
	}
}

func TestProductPreservesLoadedRecipeComponentsForRetiredMaterialOrColor(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			{
				ID:        "variant-1",
				ProductID: "prod-1",
				Name:      "Padrao",
				Slug:      "padrao",
				IsActive:  true,
				IsDefault: true,
				Filaments: []VariantFilament{
					{
						ID:                "filament-active",
						VariantID:         "variant-1",
						Material:          Material{ID: "mat-pla", Name: "PLA", Slug: "pla"},
						Color:             Color{ID: "cor-azul", Name: "Azul", Slug: "azul", HexColor: "#0066FF"},
						EstimatedWeightMg: 180000,
					},
					{
						ID:                "filament-retired",
						VariantID:         "variant-1",
						Material:          Material{ID: "mat-legado", Name: "Material legado", Slug: "material-legado"},
						Color:             Color{ID: "cor-retirada", Name: "Cor retirada", Slug: "cor-retirada", HexColor: "#CCCCCC"},
						EstimatedWeightMg: 30000,
					},
				},
			},
		}),
	})

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.SelectedVariant == nil {
		t.Fatal("expected selected variant")
	}

	if got := len(detail.SelectedVariant.Filaments); got != 2 {
		t.Fatalf("expected both recipe components to remain loaded, got %d", got)
	}

	if detail.SelectedVariant.TotalWeightMg != 210000 || detail.SelectedVariant.TotalWeightGrams != "210 g" {
		t.Fatalf("expected all recipe components in total weight, got %d %q", detail.SelectedVariant.TotalWeightMg, detail.SelectedVariant.TotalWeightGrams)
	}

	if detail.SelectedVariant.MaterialNames != "PLA, Material legado" {
		t.Fatalf("expected material names to include retired reference, got %q", detail.SelectedVariant.MaterialNames)
	}

	if detail.SelectedVariant.ColorNames != "Azul, Cor retirada" {
		t.Fatalf("expected color names to include retired reference, got %q", detail.SelectedVariant.ColorNames)
	}
}

func TestProductImageSelectionPrefersVariantImage(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			{
				ID:        "variant-1",
				ProductID: "prod-1",
				Name:      "Padrao",
				Slug:      "padrao",
				IsActive:  true,
				IsDefault: true,
				Images: []ProductImage{
					{ID: "variant-image", ProductID: "prod-1", VariantID: "variant-1", StoragePath: "produtos/variant.webp", AltText: "Variante"},
				},
			},
		}),
	}, WithSupabaseURL("https://example.supabase.co"))
	service.repository.(*fakeRepository).detail.Product.Images = []ProductImage{
		{ID: "product-image", ProductID: "prod-1", StoragePath: "produtos/product.webp", AltText: "Produto", IsPrimary: true},
	}

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(detail.Images) != 1 || detail.Images[0].ID != "variant-image" {
		t.Fatalf("expected variant image, got %#v", detail.Images)
	}
}

func TestProductImageSelectionFallsBackToProductImage(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture([]ProductVariant{
			activeVariant("padrao", "Padrao", true),
		}),
	}, WithSupabaseURL("https://example.supabase.co"))
	service.repository.(*fakeRepository).detail.Product.Images = []ProductImage{
		{ID: "product-image", ProductID: "prod-1", StoragePath: "produtos/product.webp", AltText: "Produto", IsPrimary: true},
	}

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(detail.Images) != 1 || detail.Images[0].ID != "product-image" {
		t.Fatalf("expected product image fallback, got %#v", detail.Images)
	}
}

func TestProductImageSelectionFallsBackToPlaceholderWhenURLIsUnavailable(t *testing.T) {
	service := NewService(&fakeRepository{
		detail: productDetailFixture(nil),
	})
	service.repository.(*fakeRepository).detail.Product.Images = []ProductImage{
		{ID: "product-image", ProductID: "prod-1", StoragePath: "produtos/product.webp", AltText: "Produto", IsPrimary: true},
	}

	detail, err := service.Product(context.Background(), "produto-real", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(detail.Images) != 0 {
		t.Fatalf("expected no renderable images without Supabase URL, got %#v", detail.Images)
	}
}

type fakeRepository struct {
	categories []Category
	products   []Product
	detail     ProductDetail

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

func (r *fakeRepository) GetActiveProductDetailBySlug(_ context.Context, slug string) (ProductDetail, error) {
	r.productCalls++
	r.lastSlug = slug
	if r.productErr != nil {
		return ProductDetail{}, r.productErr
	}

	return r.detail, nil
}

func productDetailFixture(variants []ProductVariant) ProductDetail {
	return ProductDetail{
		Product: Product{
			ID:               "prod-1",
			Name:             "Produto Real",
			Slug:             "produto-real",
			ShortDescription: "Objeto impresso pela PrintLab.",
			Description:      "Descricao completa do produto.",
			PriceCents:       3990,
		},
		Variants: variants,
	}
}

func activeVariant(slug string, name string, isDefault bool) ProductVariant {
	return ProductVariant{
		ID:        "id-" + slug,
		ProductID: "prod-1",
		Name:      name,
		Slug:      slug,
		IsActive:  true,
		IsDefault: isDefault,
	}
}
