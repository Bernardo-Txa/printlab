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
	ErrAuthRejected       = errors.New("admin auth rejected")
	ErrAuthUnavailable    = errors.New("admin auth unavailable")
	ErrInvalidCredentials = errors.New("admin invalid credentials")
	ErrUnauthenticated    = errors.New("admin unauthenticated")
	ErrSessionNotFound    = errors.New("admin session not found")
	ErrSessionExpired     = errors.New("admin session expired")
	ErrInvalidToken       = errors.New("admin invalid session token")
	ErrInvalidOrderID     = errors.New("admin invalid order id")
	ErrOrderNotFound      = errors.New("admin order not found")
	ErrInvalidTransition  = errors.New("admin invalid order transition")
	ErrTransitionConflict = errors.New("admin order transition conflict")
	ErrUnavailable        = errors.New("admin unavailable")
)

type AuthUser struct {
	ID string
}

type Session struct {
	ID         string
	AuthUserID string
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
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
