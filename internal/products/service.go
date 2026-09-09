package products

import (
	"context"
	"errors"
	"strings"
)

type Repository interface {
	ListActiveCategories(ctx context.Context) ([]Category, error)
	ListActiveProducts(ctx context.Context, filter ListFilter) ([]Product, error)
	GetActiveProductDetailBySlug(ctx context.Context, slug string) (ProductDetail, error)
}

type Service struct {
	repository  Repository
	supabaseURL string
}

type ServiceOption func(*Service)

func WithSupabaseURL(supabaseURL string) ServiceOption {
	return func(service *Service) {
		service.supabaseURL = strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	}
}

func NewService(repository Repository, options ...ServiceOption) *Service {
	service := &Service{repository: repository}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Catalog(ctx context.Context, filter ListFilter) (Catalog, error) {
	if s == nil || s.repository == nil {
		return Catalog{}, ErrUnavailable
	}

	if filter.CategorySlug != "" && !ValidSlug(filter.CategorySlug) {
		return Catalog{}, ErrInvalidSlug
	}

	categories, err := s.repository.ListActiveCategories(ctx)
	if err != nil {
		return Catalog{}, ErrUnavailable
	}

	products, err := s.repository.ListActiveProducts(ctx, filter)
	if err != nil {
		return Catalog{}, ErrUnavailable
	}

	for i := range products {
		s.prepareProductCard(&products[i])
	}

	return Catalog{
		Categories:           categories,
		Products:             products,
		SelectedCategorySlug: filter.CategorySlug,
		SelectedCategory:     selectedCategory(categories, filter.CategorySlug),
	}, nil
}

func (s *Service) Product(ctx context.Context, slug string, variantSlug string) (ProductDetail, error) {
	if s == nil || s.repository == nil {
		return ProductDetail{}, ErrUnavailable
	}

	if !ValidSlug(slug) {
		return ProductDetail{}, ErrInvalidSlug
	}

	if variantSlug != "" && !ValidSlug(variantSlug) {
		return ProductDetail{}, ErrInvalidSlug
	}

	detail, err := s.repository.GetActiveProductDetailBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ProductDetail{}, ErrNotFound
		}

		return ProductDetail{}, ErrUnavailable
	}

	s.prepareProductDetail(&detail, variantSlug)
	if variantSlug != "" && detail.SelectedVariant == nil {
		return ProductDetail{}, ErrNotFound
	}

	return detail, nil
}

func selectedCategory(categories []Category, slug string) *Category {
	if slug == "" {
		return nil
	}

	for i := range categories {
		if categories[i].Slug == slug {
			category := categories[i]
			return &category
		}
	}

	return nil
}

func (s *Service) prepareProductCard(product *Product) {
	if !product.HasDisplayPrice {
		product.DisplayPriceCents = product.PriceCents
		product.HasDisplayPrice = true
	}

	product.PriceBRL = FormatBRL(product.DisplayPriceCents)

	if product.PrimaryImage != nil {
		image := s.prepareImage(*product.PrimaryImage)
		product.PrimaryImage = &image
	}
}

func (s *Service) prepareProductDetail(detail *ProductDetail, variantSlug string) {
	detail.Product.PriceBRL = FormatBRL(detail.Product.PriceCents)
	detail.Product.Images = s.prepareImages(detail.Product.Images)
	detail.Product.PrimaryImage = primaryImage(detail.Product.Images)

	variants := make([]ProductVariant, 0, len(detail.Variants))
	for _, variant := range detail.Variants {
		if !variant.IsActive || variant.ProductID != detail.Product.ID {
			continue
		}

		variant.EffectivePriceCents = EffectivePriceCents(detail.Product, variant)
		variant.EffectivePriceBRL = FormatBRL(variant.EffectivePriceCents)
		variant.Filaments = prepareFilaments(variant.Filaments)
		variant.TotalWeightMg = TotalFilamentWeightMg(variant.Filaments)
		variant.TotalWeightGrams = FormatWeightGrams(variant.TotalWeightMg)
		variant.Materials = uniqueMaterials(variant.Filaments)
		variant.Colors = uniqueColors(variant.Filaments)
		variant.MaterialNames = materialNames(variant.Materials)
		variant.ColorNames = colorNames(variant.Colors)
		variant.Images = s.prepareImages(variant.Images)

		if variant.PrintTimeMinutes != nil {
			variant.PrintTimeLabel = FormatPrintTime(*variant.PrintTimeMinutes)
		}

		variants = append(variants, variant)
	}
	detail.Variants = variants

	selected := selectVariant(detail.Variants, variantSlug)
	detail.SelectedVariant = selected
	detail.CanonicalPath = "/produtos/" + detail.Product.Slug

	if selected != nil {
		detail.DisplayPriceCents = selected.EffectivePriceCents
		detail.DisplayPriceBRL = selected.EffectivePriceBRL
		if selected.PriceCents != nil {
			detail.DisplayPriceLabel = "Preco da variante"
		} else {
			detail.DisplayPriceLabel = "Preco-base"
		}
		detail.Images = displayImages(detail.Product.Images, selected)
		return
	}

	detail.DisplayPriceCents = detail.Product.PriceCents
	detail.DisplayPriceBRL = FormatBRL(detail.DisplayPriceCents)
	detail.DisplayPriceLabel = "Preco-base"
	detail.Images = displayImages(detail.Product.Images, nil)
}

func (s *Service) prepareImages(images []ProductImage) []ProductImage {
	prepared := make([]ProductImage, 0, len(images))
	for _, image := range images {
		prepared = append(prepared, s.prepareImage(image))
	}

	return prepared
}

func (s *Service) prepareImage(image ProductImage) ProductImage {
	image.URL = PublicProductImageURL(s.supabaseURL, image.StoragePath)
	return image
}

func prepareFilaments(filaments []VariantFilament) []VariantFilament {
	prepared := make([]VariantFilament, 0, len(filaments))
	for _, filament := range filaments {
		filament.EstimatedWeight = FormatWeightGrams(filament.EstimatedWeightMg)
		prepared = append(prepared, filament)
	}

	return prepared
}

func selectVariant(variants []ProductVariant, variantSlug string) *ProductVariant {
	if variantSlug != "" {
		for i := range variants {
			if variants[i].Slug == variantSlug {
				return &variants[i]
			}
		}

		return nil
	}

	for i := range variants {
		if variants[i].IsDefault {
			return &variants[i]
		}
	}

	if len(variants) > 0 {
		return &variants[0]
	}

	return nil
}

func displayImages(productImages []ProductImage, selectedVariant *ProductVariant) []ProductImage {
	if selectedVariant != nil {
		variantImages := renderableImages(selectedVariant.Images)
		if len(variantImages) > 0 {
			return variantImages
		}
	}

	return renderableImages(productImages)
}

func renderableImages(images []ProductImage) []ProductImage {
	renderable := make([]ProductImage, 0, len(images))
	for _, image := range images {
		if image.URL != "" {
			renderable = append(renderable, image)
		}
	}

	return renderable
}

func primaryImage(images []ProductImage) *ProductImage {
	for i := range images {
		if images[i].VariantID == "" && images[i].IsPrimary && images[i].URL != "" {
			return &images[i]
		}
	}

	return nil
}

func uniqueMaterials(filaments []VariantFilament) []Material {
	seen := map[string]bool{}
	var materials []Material
	for _, filament := range filaments {
		if filament.Material.ID == "" || seen[filament.Material.ID] {
			continue
		}

		seen[filament.Material.ID] = true
		materials = append(materials, filament.Material)
	}

	return materials
}

func uniqueColors(filaments []VariantFilament) []Color {
	seen := map[string]bool{}
	var colors []Color
	for _, filament := range filaments {
		if filament.Color.ID == "" || seen[filament.Color.ID] {
			continue
		}

		seen[filament.Color.ID] = true
		colors = append(colors, filament.Color)
	}

	return colors
}

func materialNames(materials []Material) string {
	names := make([]string, 0, len(materials))
	for _, material := range materials {
		names = append(names, material.Name)
	}

	return strings.Join(names, ", ")
}

func colorNames(colors []Color) string {
	names := make([]string, 0, len(colors))
	for _, color := range colors {
		names = append(names, color.Name)
	}

	return strings.Join(names, ", ")
}
