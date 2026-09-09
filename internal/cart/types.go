package cart

import (
	"errors"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const (
	CookieName      = "printlab_cart"
	TokenByteLength = 32
	HashByteLength  = 32
	MinQuantity     = 1
	MaxQuantity     = 99
	TTL             = 30 * 24 * time.Hour
)

var (
	ErrInvalidToken       = errors.New("invalid cart token")
	ErrInvalidQuantity    = errors.New("invalid cart quantity")
	ErrInvalidProduct     = errors.New("invalid cart product")
	ErrInvalidVariant     = errors.New("invalid cart variant")
	ErrProductUnavailable = errors.New("cart product unavailable")
	ErrVariantRequired    = errors.New("cart variant required")
	ErrVariantUnavailable = errors.New("cart variant unavailable")
	ErrQuantityLimit      = errors.New("cart quantity limit exceeded")
	ErrAmountOverflow     = errors.New("cart amount overflow")
	ErrNotFound           = errors.New("cart not found")
	ErrUnavailable        = errors.New("cart unavailable")
)

type Cart struct {
	ID        string
	ExpiresAt time.Time
}

type AddItemInput struct {
	ProductSlug string
	VariantSlug string
	Quantity    int
}

type ProductForAdd struct {
	ID         string
	Name       string
	Slug       string
	PriceCents int64
	IsActive   bool
	Variants   []VariantForAdd
}

type VariantForAdd struct {
	ID         string
	ProductID  string
	Name       string
	Slug       string
	PriceCents *int64
	IsActive   bool
}

type StoredItem struct {
	ID                       string
	CartID                   string
	Quantity                 int
	Product                  ProductSnapshot
	Variant                  *VariantSnapshot
	ProductImage             *products.ProductImage
	VariantImage             *products.ProductImage
	ProductHasActiveVariants bool
}

type ProductSnapshot struct {
	ID          string
	Name        string
	Slug        string
	PriceCents  int64
	IsActive    bool
	Description string
}

type VariantSnapshot struct {
	ID         string
	ProductID  string
	Name       string
	Slug       string
	PriceCents *int64
	IsActive   bool
}

type CartView struct {
	Lines               []CartLine
	SubtotalCents       int64
	SubtotalBRL         string
	IsEmpty             bool
	HasUnavailableItems bool
}

type CartLine struct {
	ID                  string
	ProductName         string
	ProductSlug         string
	VariantName         string
	VariantSlug         string
	HasVariant          bool
	Image               *products.ProductImage
	Quantity            int
	UnitPriceCents      int64
	UnitPriceBRL        string
	SubtotalCents       int64
	SubtotalBRL         string
	Available           bool
	UnavailableReason   string
	ProductAvailable    bool
	VariantAvailable    bool
	RequiresVariantPick bool
}

func EmptyView() CartView {
	return CartView{
		IsEmpty:     true,
		SubtotalBRL: products.FormatBRL(0),
	}
}
