package shipping

import (
	"errors"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
)

const (
	ProviderSuperFrete = "superfrete"
	QuoteTTL           = 30 * time.Minute
)

var (
	ErrUnavailable            = errors.New("shipping unavailable")
	ErrNotConfigured          = errors.New("shipping not configured")
	ErrCartRequired           = errors.New("shipping cart required")
	ErrEmptyCart              = errors.New("shipping cart empty")
	ErrUnavailableItems       = errors.New("shipping cart has unavailable items")
	ErrDetailsRequired        = errors.New("shipping customer details required")
	ErrMissingShippingProfile = errors.New("shipping profile missing")
	ErrNoShippingBoxes        = errors.New("shipping boxes unavailable")
	ErrNoFittingBox           = errors.New("shipping box not found")
	ErrNoQuotes               = errors.New("shipping quotes unavailable")
	ErrInvalidService         = errors.New("shipping service invalid")
	ErrAmountOverflow         = errors.New("shipping amount overflow")
	ErrInvalidPackage         = errors.New("shipping package invalid")
)

type DimensionsMM struct {
	Height int
	Width  int
	Length int
}

type ShippingProfile struct {
	WeightG    int64
	Dimensions DimensionsMM
}

type CartItem struct {
	ID             string
	ProductID      string
	ProductName    string
	VariantID      string
	VariantName    string
	Quantity       int
	ProductProfile *ShippingProfile
	VariantProfile *ShippingProfile
}

type ShippingBox struct {
	ID               string
	Name             string
	Slug             string
	Internal         DimensionsMM
	External         DimensionsMM
	PackagingWeightG int64
	SortOrder        int
}

type ShippingPackage struct {
	Box        ShippingBox
	WeightG    int64
	Dimensions DimensionsMM
	InputHash  []byte
}

type ShippingQuote struct {
	Provider         string
	ServiceCode      string
	ServiceName      string
	CarrierName      string
	PriceCents       int64
	PriceBRL         string
	DeliveryTimeDays *int
	DeliveryTime     string
	Selected         bool
}

type ShippingSelection struct {
	CartID           string
	ShippingBoxID    string
	Provider         string
	ServiceCode      string
	ServiceName      string
	CarrierName      string
	PriceCents       int64
	DeliveryTimeDays *int
	PackageWeightG   int64
	PackageHeightMM  int
	PackageWidthMM   int
	PackageLengthMM  int
	InputHash        []byte
	QuotedAt         time.Time
	ExpiresAt        time.Time
}

type PreparedQuote struct {
	Cart      cartdomain.Cart
	CartView  cartdomain.CartView
	Package   ShippingPackage
	Quotes    []ShippingQuote
	Selection *ShippingSelection
}

type CheckoutShippingPage struct {
	Cart                cartdomain.CartView
	Quotes              []ShippingQuote
	ProductsSubtotalBRL string
	ShippingPriceBRL    string
	PartialTotalBRL     string
	Selected            bool
	Message             string
	Unavailable         bool
}

type SelectResult struct {
	Page CheckoutShippingPage
}

type QuoteProduct struct {
	ID        string
	ProductID string
	VariantID string
	Quantity  int
	Profile   ShippingProfile
}

type QuoteFingerprint struct {
	OriginPostalCode      string
	DestinationPostalCode string
	Services              []string
	Options               QuoteOptions
	Products              []QuoteProductFingerprint
	Box                   QuoteBoxFingerprint
}

type QuoteOptions struct {
	OwnHand           bool
	Receipt           bool
	UseInsuranceValue bool
}

type QuoteProductFingerprint struct {
	ID        string
	ProductID string
	VariantID string
	Quantity  int
	WeightG   int64
	HeightMM  int
	WidthMM   int
	LengthMM  int
}

type QuoteBoxFingerprint struct {
	ID               string
	ExternalHeightMM int
	ExternalWidthMM  int
	ExternalLengthMM int
	PackagingWeightG int64
}
