package payments

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const (
	ProviderInfinitePay = "infinitepay"

	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"

	PaymentStatusPending = "pending"
	PaymentStatusPaid    = "paid"

	ReturnStatusPending     = "pending"
	ReturnStatusConfirmed   = "confirmed"
	ReturnStatusUnavailable = "unavailable"

	checkoutHostBR      = "checkout.infinitepay.com.br"
	checkoutHostCurrent = "checkout.infinitepay.io"
)

var (
	ErrInvalidOrderID       = errors.New("invalid payment order id")
	ErrOrderNotFound        = errors.New("payment order not found")
	ErrOrderNotPayable      = errors.New("payment order not payable")
	ErrOrderAlreadyPaid     = errors.New("payment order already paid")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentNotConfigured = errors.New("payment not configured")
	ErrInvalidReturn        = errors.New("invalid payment return")
	ErrProviderUnavailable  = errors.New("payment provider unavailable")
	ErrInvalidCheckoutURL   = errors.New("invalid checkout url")
	ErrAmountMismatch       = errors.New("payment amount mismatch")
	ErrAmountOverflow       = errors.New("payment amount overflow")
	ErrUnavailable          = errors.New("payments unavailable")
)

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	providerIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,200}$`)
)

type CheckoutOrder struct {
	ID          string
	OrderNSU    string
	OrderNumber int64
	Status      string
	TotalCents  int64
	Items       []CheckoutOrderItem
	Customer    CheckoutCustomer
	Address     CheckoutAddress
	Shipping    CheckoutShipping
}

type CheckoutOrderItem struct {
	ProductName    string
	VariantName    string
	Quantity       int
	UnitPriceCents int64
}

type CheckoutCustomer struct {
	Name  string
	Email string
	Phone string
}

type CheckoutAddress struct {
	PostalCode   string
	Street       string
	Number       string
	Complement   string
	Neighborhood string
}

type CheckoutShipping struct {
	ServiceName string
	PriceCents  int64
}

type CheckoutRequest struct {
	Handle      string
	RedirectURL string
	OrderNSU    string
	Items       []CheckoutItem
	Customer    CheckoutCustomer
	Address     CheckoutAddress
}

type CheckoutItem struct {
	Quantity    int
	PriceCents  int64
	Description string
}

type CheckoutCreated struct {
	URL string
}

type CheckoutStartResult struct {
	OrderID     string
	CheckoutURL string
	Reused      bool
}

type PaymentCheckRequest struct {
	Handle         string
	OrderNSU       string
	TransactionNSU string
	Slug           string
}

type PaymentCheckResult struct {
	Success         bool
	Paid            bool
	AmountCents     int64
	PaidAmountCents int64
	Installments    *int
	CaptureMethod   string
}

type PaymentVerificationTarget struct {
	OrderID       string
	OrderNumber   int64
	OrderStatus   string
	PaymentStatus string
	TotalCents    int64
}

type VerifiedPayment struct {
	InvoiceSlug     string
	TransactionNSU  string
	AmountCents     int64
	PaidAmountCents int64
	Installments    *int
	CaptureMethod   string
}

type ReturnInput struct {
	OrderNSU       string
	TransactionNSU string
	Slug           string
}

type ReturnResult struct {
	Status  string
	OrderID string
	Message string
}

type ReturnPage struct {
	Title            string
	Message          string
	OrderID          string
	CanReturnToOrder bool
}

type OrderPaymentView struct {
	CanPay             bool
	ShowUnavailable    bool
	ShowConfirmed      bool
	Message            string
	UnavailableMessage string
}

func CanonicalOrderNSU(orderID string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(orderID))
	if !uuidPattern.MatchString(value) {
		return "", false
	}

	return value, true
}

func ValidOrderID(orderID string) bool {
	_, ok := CanonicalOrderNSU(orderID)
	return ok
}

func NormalizeReturnInput(input ReturnInput) (ReturnInput, error) {
	orderNSU, ok := CanonicalOrderNSU(input.OrderNSU)
	if !ok {
		return ReturnInput{}, ErrInvalidReturn
	}

	transactionNSU := strings.TrimSpace(input.TransactionNSU)
	if !providerIDPattern.MatchString(transactionNSU) {
		return ReturnInput{}, ErrInvalidReturn
	}

	slug := strings.TrimSpace(input.Slug)
	if !providerIDPattern.MatchString(slug) {
		return ReturnInput{}, ErrInvalidReturn
	}

	return ReturnInput{
		OrderNSU:       orderNSU,
		TransactionNSU: transactionNSU,
		Slug:           slug,
	}, nil
}

func PaymentRedirectURL(siteURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(siteURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if parsed.Scheme != "https" {
		return "", false
	}

	parsed.Path = "/pagamento/retorno"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), true
}

func ValidateCheckoutURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Port() != "" || !isAllowedCheckoutHost(parsed.Hostname()) {
		return ErrInvalidCheckoutURL
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return ErrInvalidCheckoutURL
	}

	return nil
}

func isAllowedCheckoutHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case checkoutHostBR, checkoutHostCurrent:
		return true
	default:
		return false
	}
}

func ReturnPageFor(result ReturnResult) ReturnPage {
	switch result.Status {
	case ReturnStatusPending:
		return ReturnPage{
			Title:            "Pagamento em analise",
			Message:          "Pagamento ainda nao foi confirmado.",
			OrderID:          result.OrderID,
			CanReturnToOrder: result.OrderID != "",
		}
	case ReturnStatusUnavailable:
		return ReturnPage{
			Title:            "Pagamento nao confirmado",
			Message:          "Nao foi possivel confirmar o pagamento agora. Tente novamente.",
			OrderID:          result.OrderID,
			CanReturnToOrder: result.OrderID != "",
		}
	default:
		return ReturnPage{
			Title:            "Pagamento nao confirmado",
			Message:          "Nao foi possivel confirmar o pagamento agora. Tente novamente.",
			OrderID:          result.OrderID,
			CanReturnToOrder: result.OrderID != "",
		}
	}
}
