package products

import (
	"context"
	"errors"
)

type Repository interface {
	ListActiveCategories(ctx context.Context) ([]Category, error)
	ListActiveProducts(ctx context.Context, filter ListFilter) ([]Product, error)
	GetActiveProductBySlug(ctx context.Context, slug string) (Product, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
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
		products[i].PriceBRL = FormatBRL(products[i].PriceCents)
	}

	return Catalog{
		Categories:           categories,
		Products:             products,
		SelectedCategorySlug: filter.CategorySlug,
		SelectedCategory:     selectedCategory(categories, filter.CategorySlug),
	}, nil
}

func (s *Service) Product(ctx context.Context, slug string) (Product, error) {
	if s == nil || s.repository == nil {
		return Product{}, ErrUnavailable
	}

	if !ValidSlug(slug) {
		return Product{}, ErrInvalidSlug
	}

	product, err := s.repository.GetActiveProductBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Product{}, ErrNotFound
		}

		return Product{}, ErrUnavailable
	}

	product.PriceBRL = FormatBRL(product.PriceCents)

	return product, nil
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
