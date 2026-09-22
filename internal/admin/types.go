package admin

import (
	"errors"
	"time"
)

const (
	CookieName      = "printlab_admin_session"
	TokenByteLength = 32
	HashByteLength  = 32
	SessionTTL      = 8 * time.Hour

	InvalidCredentialsMessage = "E-mail ou senha inválidos."
)

var (
	ErrAuthRejected        = errors.New("admin auth rejected")
	ErrAuthUnavailable     = errors.New("admin auth unavailable")
	ErrAuthConfiguration   = errors.New("admin auth configuration invalid")
	ErrAuthInvalidResponse = errors.New("admin auth response invalid")
	ErrAuthRateLimited     = errors.New("admin auth rate limited")
	ErrMFAInvalidCode      = errors.New("admin mfa invalid code")
	ErrMFAFactor           = errors.New("admin mfa factor invalid")
	ErrInvalidCredentials  = errors.New("admin invalid credentials")
	ErrUnauthenticated     = errors.New("admin unauthenticated")
	ErrSessionNotFound     = errors.New("admin session not found")
	ErrSessionExpired      = errors.New("admin session expired")
	ErrInvalidToken        = errors.New("admin invalid session token")
	ErrInvalidOrderID      = errors.New("admin invalid order id")
	ErrOrderNotFound       = errors.New("admin order not found")
	ErrInvalidTransition   = errors.New("admin invalid order transition")
	ErrTransitionConflict  = errors.New("admin order transition conflict")
	ErrInvalidCatalogID    = errors.New("admin invalid catalog id")
	ErrCatalogNotFound     = errors.New("admin catalog record not found")
	ErrValidation          = errors.New("admin validation failed")
	ErrDuplicateSlug       = errors.New("admin duplicate slug")
	ErrDuplicateSKU        = errors.New("admin duplicate sku")
	ErrUnavailable         = errors.New("admin unavailable")
	ErrStorageUnavailable  = errors.New("admin storage unavailable")
	ErrInvalidImageType    = errors.New("admin invalid image type")
	ErrImageTooLarge       = errors.New("admin image too large")
	ErrInvalidImagePath    = errors.New("admin invalid image path")
)

type AuthUser struct {
	ID      string      `json:"id"`
	Factors []MFAFactor `json:"factors"`
}

// Refresh tokens are intentionally not decoded or retained.
type AuthSession struct {
	AccessToken string   `json:"access_token"`
	User        AuthUser `json:"user"`
}

type MFAFactor struct {
	ID           string `json:"id"`
	Type         string `json:"factor_type"`
	Status       string `json:"status"`
	FriendlyName string `json:"friendly_name"`
}

type TOTPEnrollment struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	TOTP struct {
		QRCode string `json:"qr_code"`
		Secret string `json:"secret"`
	} `json:"totp"`
}

type MFAPage struct {
	Setup    bool
	Factors  []MFAFactor
	FactorID string
	QRCode   string
	Secret   string
	Message  string
}

type Session struct {
	ID            string
	AuthUserID    string
	TokenHash     []byte
	CreatedAt     time.Time
	ExpiresAt     time.Time
	MFAVerifiedAt *time.Time
}

type LoginResult struct {
	PendingToken string
	NextPath     string
	Token        string
	ExpiresAt    time.Time
}

type Dashboard struct {
	PendingPayment        int
	PaidWaitingProduction int
	InProduction          int
	WaitingShipment       int
}

type OrderStatusSnapshot struct {
	OrderStatus      string
	ProductionStatus string
	ShippingStatus   string
}

type OrderListFilter struct {
	Status string
	Page   int
}

type OrderListPage struct {
	Orders       []OrderListItem
	Status       string
	StatusLabel  string
	Page         int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
	Options      []OrderListOption
}

type OrderListOption struct {
	Value  string
	Label  string
	URL    string
	Active bool
}

type OrderListItem struct {
	ID                    string
	OrderNumber           int64
	OrderNumberLabel      string
	OrderStatus           string
	OrderStatusLabel      string
	ProductionStatus      string
	ProductionStatusLabel string
	ShippingStatus        string
	ShippingStatusLabel   string
	CreatedAt             time.Time
	CreatedAtLabel        string
	TotalCents            int64
	TotalBRL              string
	ItemCount             int
	UnitCount             int
	DetailURL             string
}

type OrderDetail struct {
	ID                    string
	PublicTrackingID      string
	TrackingURL           string
	OrderNumber           int64
	OrderNumberLabel      string
	OrderStatus           string
	OrderStatusLabel      string
	ProductionStatus      string
	ProductionStatusLabel string
	ShippingStatus        string
	ShippingStatusLabel   string
	CreatedAt             time.Time
	CreatedAtLabel        string
	ProductsSubtotalCents int64
	ProductsSubtotalBRL   string
	ShippingPriceCents    int64
	ShippingPriceBRL      string
	TotalCents            int64
	TotalBRL              string
	Customer              OrderCustomer
	Address               OrderAddress
	HasAddress            bool
	Shipping              OrderShipping
	Payment               OrderPayment
	Items                 []OrderDetailItem
	Events                []OrderEvent
	ProductionActions     []OrderAction
	ShippingActions       []OrderAction
}

type OrderCustomer struct {
	FullName string
	Email    string
	Phone    string
	CPF      string
}

type OrderAddress struct {
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

type OrderShipping struct {
	DeliveryMethod     string
	Provider           string
	ServiceCode        string
	ServiceName        string
	CarrierName        string
	DeliveryTime       string
	ShippingBoxName    string
	PriceBRL           string
	PackageWeightG     int64
	PackageWeightLabel string
	PackageHeightMM    int
	PackageWidthMM     int
	PackageLengthMM    int
	DimensionsLabel    string
	QuotedAt           *time.Time
	QuotedAtLabel      string
}

type OrderPayment struct {
	Available         bool
	Status            string
	StatusLabel       string
	CaptureMethod     string
	PaidAt            *time.Time
	PaidAtLabel       string
	AmountBRL         string
	PaidAmountBRL     string
	Installments      *int
	InstallmentsLabel string
}

type OrderDetailItem struct {
	ColorID                          string
	ColorName                        string
	ColorSlug                        string
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
	Filaments                        []OrderItemFilament
}

type OrderItemFilament struct {
	MaterialName             string
	ColorName                string
	HexColor                 string
	EstimatedWeightMgPerUnit int64
	EstimatedWeightLabel     string
	Label                    string
}

type OrderEvent struct {
	EventType       string
	EventTypeLabel  string
	FromStatus      string
	FromStatusLabel string
	ToStatus        string
	ToStatusLabel   string
	CreatedAt       time.Time
	CreatedAtLabel  string
}

type OrderAction struct {
	Status string
	Label  string
}
