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
	ID                string
	Category          *Category
	Name              string
	Slug              string
	ShortDescription  string
	Description       string
	PriceCents        int64
	DisplayPriceCents int64
	HasDisplayPrice   bool
	PriceBRL          string
	PriceFrom         bool
	IsFeatured        bool
	PrimaryImage      *ProductImage
	Images            []ProductImage
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

type ProductVariant struct {
	ID                  string
	ProductID           string
	Name                string
	Slug                string
	SKU                 string
	PriceCents          *int64
	EffectivePriceCents int64
	EffectivePriceBRL   string
	IsActive            bool
	IsDefault           bool
	SortOrder           int
	PrintTimeMinutes    *int
	PrintTimeLabel      string
	Filaments           []VariantFilament
	Materials           []Material
	Colors              []Color
	MaterialNames       string
	ColorNames          string
	TotalWeightMg       int64
	TotalWeightGrams    string
	Images              []ProductImage
}

type Material struct {
	ID          string
	Name        string
	Slug        string
	Description string
}

type Color struct {
	ID       string
	Name     string
	Slug     string
	HexColor string
}

type VariantFilament struct {
	ID                string
	VariantID         string
	Material          Material
	Color             Color
	EstimatedWeightMg int64
	EstimatedWeight   string
	Label             string
	SortOrder         int
}

type ProductImage struct {
	ID          string
	ProductID   string
	VariantID   string
	StoragePath string
	URL         string
	AltText     string
	SortOrder   int
	IsPrimary   bool
}

type ProductDetail struct {
	Product           Product
	Variants          []ProductVariant
	SelectedVariant   *ProductVariant
	Images            []ProductImage
	DisplayPriceCents int64
	DisplayPriceBRL   string
	DisplayPriceLabel string
	CanonicalPath     string
}
