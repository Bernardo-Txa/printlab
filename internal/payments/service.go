package payments

import (
	"context"
	"log"
	"math"
	"strings"
	"time"
)

type CheckoutCreator func(ctx context.Context, order CheckoutOrder) (CheckoutCreated, error)

type Repository interface {
	CreateOrReuseCheckout(ctx context.Context, orderID string, create CheckoutCreator) (CheckoutStartResult, error)
	PaymentTargetByOrderNSU(ctx context.Context, orderNSU string) (PaymentVerificationTarget, error)
	MarkPaid(ctx context.Context, orderNSU string, payment VerifiedPayment, paidAt time.Time) (ReturnResult, error)
}

type Service struct {
	repository  Repository
	gateway     Gateway
	handle      string
	redirectURL string
	webhookURL  string
	now         func() time.Time
}

type ServiceOption func(*Service)

func WithClock(now func() time.Time) ServiceOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func NewService(repository Repository, gateway Gateway, handle string, siteURL string, options ...ServiceOption) *Service {
	redirectURL, _ := PaymentRedirectURL(siteURL)
	webhookURL, _ := PaymentWebhookURL(siteURL)
	service := &Service{
		repository:  repository,
		gateway:     gateway,
		handle:      strings.TrimSpace(handle),
		redirectURL: redirectURL,
		webhookURL:  webhookURL,
		now:         time.Now,
	}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Available() bool {
	return s != nil &&
		s.repository != nil &&
		s.gateway != nil &&
		s.handle != "" &&
		s.redirectURL != "" &&
		s.webhookURL != ""
}

func (s *Service) StartCheckout(ctx context.Context, orderID string) (CheckoutStartResult, error) {
	canonicalOrderID, ok := CanonicalOrderNSU(orderID)
	if !ok {
		return CheckoutStartResult{}, ErrInvalidOrderID
	}
	if !s.Available() {
		return CheckoutStartResult{OrderID: canonicalOrderID}, ErrPaymentNotConfigured
	}

	return s.repository.CreateOrReuseCheckout(ctx, canonicalOrderID, func(ctx context.Context, order CheckoutOrder) (CheckoutCreated, error) {
		request, err := BuildCheckoutRequest(order, s.handle, s.redirectURL, s.webhookURL)
		if err != nil {
			return CheckoutCreated{}, err
		}

		return s.gateway.CreateCheckout(ctx, request)
	})
}

func (s *Service) ConfirmReturn(ctx context.Context, input ReturnInput) (ReturnResult, error) {
	normalized, err := NormalizeReturnInput(input)
	if err != nil {
		return ReturnResult{Status: ReturnStatusUnavailable}, err
	}
	if !s.Available() {
		return ReturnResult{Status: ReturnStatusUnavailable}, ErrPaymentNotConfigured
	}

	return s.confirmPayment(ctx, normalized, paymentConfirmationOptions{
		pendingAsError:    false,
		amountMismatchLog: "payment amount mismatch",
	})
}

func (s *Service) ConfirmWebhook(ctx context.Context, input WebhookInput) (ReturnResult, error) {
	normalized, err := NormalizeWebhookInput(input)
	if err != nil {
		return ReturnResult{Status: ReturnStatusUnavailable}, err
	}
	if !s.Available() {
		return ReturnResult{Status: ReturnStatusUnavailable}, ErrPaymentNotConfigured
	}

	return s.confirmPayment(ctx, normalized, paymentConfirmationOptions{
		pendingAsError:    true,
		amountMismatchLog: "payment webhook amount mismatch",
	})
}

type paymentConfirmationOptions struct {
	pendingAsError    bool
	amountMismatchLog string
}

func (s *Service) confirmPayment(ctx context.Context, normalized ReturnInput, options paymentConfirmationOptions) (ReturnResult, error) {
	target, err := s.repository.PaymentTargetByOrderNSU(ctx, normalized.OrderNSU)
	if err != nil {
		return ReturnResult{Status: ReturnStatusUnavailable}, err
	}
	if target.OrderStatus == OrderStatusPaid || target.PaymentStatus == PaymentStatusPaid {
		return ReturnResult{Status: ReturnStatusConfirmed, OrderID: target.OrderID, Message: ReturnMessageAlreadyPaid}, nil
	}

	check, err := s.gateway.CheckPayment(ctx, PaymentCheckRequest{
		Handle:         s.handle,
		OrderNSU:       normalized.OrderNSU,
		TransactionNSU: normalized.TransactionNSU,
		Slug:           normalized.Slug,
	})
	if err != nil {
		return ReturnResult{Status: ReturnStatusUnavailable, OrderID: target.OrderID}, err
	}
	if !check.Success || !check.Paid {
		result := ReturnResult{Status: ReturnStatusPending, OrderID: target.OrderID}
		if options.pendingAsError {
			return result, ErrPaymentNotConfirmed
		}

		return result, nil
	}
	if check.AmountCents != target.TotalCents {
		log.Printf("%s order_id=%s order_number=%d", options.amountMismatchLog, target.OrderID, target.OrderNumber)
		return ReturnResult{Status: ReturnStatusUnavailable, OrderID: target.OrderID}, ErrAmountMismatch
	}

	result, err := s.repository.MarkPaid(ctx, normalized.OrderNSU, VerifiedPayment{
		InvoiceSlug:     normalized.Slug,
		TransactionNSU:  normalized.TransactionNSU,
		AmountCents:     check.AmountCents,
		PaidAmountCents: check.PaidAmountCents,
		Installments:    check.Installments,
		CaptureMethod:   check.CaptureMethod,
	}, s.now())
	if err != nil {
		return result, err
	}
	if result.Status == ReturnStatusConfirmed && result.Message == "" {
		result.Message = ReturnMessageVerified
	}

	return result, nil
}

func BuildCheckoutRequest(order CheckoutOrder, handle string, redirectURL string, webhookURL string) (CheckoutRequest, error) {
	orderNSU := order.OrderNSU
	if orderNSU == "" {
		var ok bool
		orderNSU, ok = CanonicalOrderNSU(order.ID)
		if !ok {
			return CheckoutRequest{}, ErrInvalidOrderID
		}
	}
	if order.Status == OrderStatusPaid {
		return CheckoutRequest{}, ErrOrderAlreadyPaid
	}
	if order.Status != OrderStatusPendingPayment {
		return CheckoutRequest{}, ErrOrderNotPayable
	}

	request := CheckoutRequest{
		Handle:      strings.TrimSpace(handle),
		RedirectURL: strings.TrimSpace(redirectURL),
		WebhookURL:  strings.TrimSpace(webhookURL),
		OrderNSU:    orderNSU,
		Customer: CheckoutCustomer{
			Name:  strings.TrimSpace(order.Customer.Name),
			Email: strings.TrimSpace(order.Customer.Email),
			Phone: strings.TrimSpace(order.Customer.Phone),
		},
		Address: CheckoutAddress{
			PostalCode:   strings.TrimSpace(order.Address.PostalCode),
			Street:       strings.TrimSpace(order.Address.Street),
			Number:       strings.TrimSpace(order.Address.Number),
			Complement:   strings.TrimSpace(order.Address.Complement),
			Neighborhood: strings.TrimSpace(order.Address.Neighborhood),
		},
	}

	var expectedTotal int64
	for _, item := range order.Items {
		if item.Quantity <= 0 || item.UnitPriceCents < 0 {
			return CheckoutRequest{}, ErrAmountMismatch
		}
		lineTotal, err := multiplyCents(item.UnitPriceCents, item.Quantity)
		if err != nil {
			return CheckoutRequest{}, err
		}
		expectedTotal, err = addCents(expectedTotal, lineTotal)
		if err != nil {
			return CheckoutRequest{}, err
		}
		request.Items = append(request.Items, CheckoutItem{
			Quantity:    item.Quantity,
			PriceCents:  item.UnitPriceCents,
			Description: checkoutItemDescription(item),
		})
	}

	if order.Shipping.PriceCents > 0 {
		var err error
		expectedTotal, err = addCents(expectedTotal, order.Shipping.PriceCents)
		if err != nil {
			return CheckoutRequest{}, err
		}
		request.Items = append(request.Items, CheckoutItem{
			Quantity:    1,
			PriceCents:  order.Shipping.PriceCents,
			Description: checkoutShippingDescription(order.Shipping.ServiceName),
		})
	}

	if expectedTotal != order.TotalCents {
		return CheckoutRequest{}, ErrAmountMismatch
	}
	if len(request.Items) == 0 {
		return CheckoutRequest{}, ErrAmountMismatch
	}

	return request, nil
}

func checkoutItemDescription(item CheckoutOrderItem) string {
	description := strings.TrimSpace(item.ProductName)
	if variant := strings.TrimSpace(item.VariantName); variant != "" {
		description += " - " + variant
	}
	if description == "" {
		return "Produto PrintLab"
	}

	return description
}

func checkoutShippingDescription(serviceName string) string {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return "Frete"
	}

	return "Frete - " + serviceName
}

func multiplyCents(value int64, quantity int) (int64, error) {
	if value < 0 || quantity < 1 {
		return 0, ErrAmountOverflow
	}
	if value > math.MaxInt64/int64(quantity) {
		return 0, ErrAmountOverflow
	}

	return value * int64(quantity), nil
}

func addCents(left int64, right int64) (int64, error) {
	if left < 0 || right < 0 || left > math.MaxInt64-right {
		return 0, ErrAmountOverflow
	}

	return left + right, nil
}
