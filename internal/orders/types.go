package orders

import (
	"errors"
	"time"
)

const (
	StatusPendingPayment = "pending_payment"
	StatusPaid           = "paid"
	CurrencyBRL          = "BRL"
	StaleReviewMessage   = "Algumas informações da sua compra foram atualizadas. Revise os dados antes de confirmar."

	ProductionStatusWaiting      = "waiting"
	ProductionStatusInProduction = "in_production"
	ProductionStatusCompleted    = "completed"

	ShippingStatusWaiting   = "waiting"
	ShippingStatusPreparing = "preparing"
	ShippingStatusShipped   = "shipped"
	ShippingStatusDelivered = "delivered"
)

var (
	ErrCartRequired      = errors.New("order cart required")
	ErrEmptyCart         = errors.New("order cart empty")
	ErrUnavailableItems  = errors.New("order cart has unavailable items")
	ErrDetailsRequired   = errors.New("order checkout details required")
	ErrShippingRequired  = errors.New("order shipping required")
	ErrShippingExpired   = errors.New("order shipping expired")
	ErrShippingChanged   = errors.New("order shipping changed")
	ErrStaleReview       = errors.New("order review stale")
	ErrInvalidOrderID    = errors.New("invalid order id")
	ErrInvalidTrackingID = errors.New("invalid order tracking id")
	ErrNotFound          = errors.New("order not found")
	ErrAmountOverflow    = errors.New("order amount overflow")
	ErrUnavailable       = errors.New("orders unavailable")
)

type ReviewParams struct {
	OriginPostalCode string
	ServiceCodes     []string
}

type ReviewPage struct {
	Items                  []ReviewItem
	Customer               ReviewCustomer
	Address                ReviewAddress
	Shipping               ReviewShipping
	ProductsSubtotalCents  int64
	ProductsSubtotalBRL    string
	ShippingPriceCents     int64
	ShippingPriceBRL       string
	TotalCents             int64
	TotalBRL               string
	Fingerprint            string
	Message                string
	HasProductionSnapshots bool
}

type ReviewItem struct {
	CartItemID                       string
	ProductID                        string
	VariantID                        string
	ProductName                      string
	ProductSlug                      string
	VariantName                      string
	VariantSlug                      string
	SKU                              string
	HasVariant                       bool
	Quantity                         int
	UnitPriceCents                   int64
	UnitPriceBRL                     string
	LineTotalCents                   int64
	LineTotalBRL                     string
	UnitPrintTimeMinutes             *int
	UnitEstimatedFilamentWeightMg    *int64
	UnitEstimatedFilamentWeightLabel string
	Filaments                        []ReviewItemFilament
	SortOrder                        int
}

type ReviewItemFilament struct {
	MaterialName             string
	MaterialSlug             string
	ColorName                string
	ColorSlug                string
	HexColor                 string
	EstimatedWeightMgPerUnit int64
	EstimatedWeightLabel     string
	Label                    string
	SortOrder                int
}

type ReviewCustomer struct {
	FullName  string
	Email     string
	Phone     string
	CPF       string
	MaskedCPF string
}

type ReviewAddress struct {
	PostalCode  string
	Street      string
	Number      string
	Complement  string
	District    string
	City        string
	State       string
	CountryCode string
	LineOne     string
	LineTwo     string
}

type ReviewShipping struct {
	Provider         string
	ServiceCode      string
	ServiceName      string
	CarrierName      string
	DeliveryTimeDays *int
	DeliveryTime     string
	ShippingBoxName  string
	PriceCents       int64
	PriceBRL         string
	PackageWeightG   int64
	PackageHeightMM  int
	PackageWidthMM   int
	PackageLengthMM  int
	QuotedAt         *time.Time
}

type ConfirmResult struct {
	OrderID      string
	OrderNumber  int64
	Status       string
	Page         ReviewPage
	ExpireCookie bool
}

type OrderPage struct {
	ID                  string
	PublicTrackingID    string
	TrackingURL         string
	OrderNumber         int64
	OrderNumberLabel    string
	Status              string
	StatusLabel         string
	CreatedAt           time.Time
	Items               []OrderItem
	Shipping            OrderShipping
	ProductsSubtotalBRL string
	ShippingPriceBRL    string
	TotalBRL            string
}

type OrderItem struct {
	ProductName                      string
	VariantName                      string
	SKU                              string
	HasVariant                       bool
	Quantity                         int
	UnitPriceBRL                     string
	LineTotalBRL                     string
	UnitPrintTimeMinutes             *int
	UnitEstimatedFilamentWeightMg    *int64
	UnitEstimatedFilamentWeightLabel string
	Filaments                        []ReviewItemFilament
}

type OrderShipping struct {
	ServiceName      string
	CarrierName      string
	DeliveryTimeDays *int
	DeliveryTime     string
	ShippingBoxName  string
	PriceBRL         string
	PackageWeightG   int64
	PackageHeightMM  int
	PackageWidthMM   int
	PackageLengthMM  int
}

type TrackingRecord struct {
	OrderNumber      int64
	Status           string
	CreatedAt        time.Time
	ProductionStatus string
	ShippingStatus   string
	ServiceName      string
	CarrierName      string
}

type TrackingPage struct {
	OrderNumber           int64
	OrderNumberLabel      string
	CreatedAt             time.Time
	CreatedAtLabel        string
	CreatedAtISO          string
	PaymentStatus         string
	PaymentStatusLabel    string
	ProductionStatus      string
	ProductionStatusLabel string
	ShippingStatus        string
	ShippingStatusLabel   string
	ShippingServiceLabel  string
	Steps                 []TrackingStep
}

type TrackingStep struct {
	Title       string
	StatusLabel string
	Description string
}
