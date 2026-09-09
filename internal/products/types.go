package products

import "errors"

var (
	ErrInvalidSlug = errors.New("invalid slug")
	ErrNotFound    = errors.New("product not found")
	ErrUnavailable = errors.New("catalog unavailable")
)

type Category struct {
	ID          string
	Name        string
	Slug        string
	Description string
}

type Product struct {
	ID               string
	Category         *Category
	Name             string
	Slug             string
	ShortDescription string
	Description      string
	PriceCents       int64
	PriceBRL         string
	IsFeatured       bool
}

type ListFilter struct {
	CategorySlug string
}

type Catalog struct {
	Categories           []Category
	Products             []Product
	SelectedCategorySlug string
	SelectedCategory     *Category
}
