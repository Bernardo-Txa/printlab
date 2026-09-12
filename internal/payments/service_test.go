package payments

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testOrderID = "22222222-2222-2222-2222-222222222222"

func TestBuildCheckoutRequestUsesOrderSnapshotAndUnitPrices(t *testing.T) {
	request, err := BuildCheckoutRequest(checkoutOrderFixture(), "printlab", "https://printlab.example/pagamento/retorno", "https://printlab.example/webhooks/infinitepay")
	if err != nil {
		t.Fatalf("expected checkout request, got %v", err)
	}

	if request.Handle != "printlab" || request.RedirectURL != "https://printlab.example/pagamento/retorno" || request.WebhookURL != "https://printlab.example/webhooks/infinitepay" || request.OrderNSU != testOrderID {
		t.Fatalf("unexpected checkout request identity fields: %#v", request)
	}
	if len(request.Items) != 2 {
		t.Fatalf("expected product and shipping items, got %#v", request.Items)
	}
	if request.Items[0].Quantity != 3 || request.Items[0].PriceCents != 1990 {
		t.Fatalf("expected unit price 1990 with quantity 3, got %#v", request.Items[0])
	}
	if request.Items[0].Description != "Produto Real - Padrao" {
		t.Fatalf("expected product and variant description, got %q", request.Items[0].Description)
	}
	if request.Items[1].Quantity != 1 || request.Items[1].PriceCents != 1000 || request.Items[1].Description != "Frete - SEDEX" {
		t.Fatalf("expected shipping line item, got %#v", request.Items[1])
	}
	if request.Customer.Name != "Joao Silva" || request.Customer.Email != "joao@example.com" || request.Customer.Phone != "+5527999999999" {
		t.Fatalf("expected customer snapshot, got %#v", request.Customer)
	}
	if request.Address.PostalCode != "29100000" || request.Address.Street != "Rua Um" || request.Address.Neighborhood != "Centro" || request.Address.Number != "12A" || request.Address.Complement != "Apto 302" {
		t.Fatalf("expected documented address fields, got %#v", request.Address)
	}
}

func TestStartCheckoutDoesNotCallProviderOnTotalMismatch(t *testing.T) {
	order := checkoutOrderFixture()
	order.TotalCents = 9999
	repository := &fakePaymentRepository{checkoutOrder: order}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	_, err := service.StartCheckout(context.Background(), testOrderID)
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("expected ErrAmountMismatch, got %v", err)
	}
	if gateway.createCalls != 0 {
		t.Fatal("expected provider not to be called on total mismatch")
	}
}

func TestStartCheckoutSendsServerGeneratedWebhookURL(t *testing.T) {
	repository := &fakePaymentRepository{}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	_, err := service.StartCheckout(context.Background(), testOrderID)
	if err != nil {
		t.Fatalf("expected checkout start, got %v", err)
	}
	if gateway.createCalls != 1 {
		t.Fatalf("expected provider to be called once, got %d", gateway.createCalls)
	}
	if gateway.lastRequest.RedirectURL != "https://printlab.example/pagamento/retorno" {
		t.Fatalf("expected server redirect URL, got %q", gateway.lastRequest.RedirectURL)
	}
	if gateway.lastRequest.WebhookURL != "https://printlab.example/webhooks/infinitepay" {
		t.Fatalf("expected server webhook URL, got %q", gateway.lastRequest.WebhookURL)
	}
}

func TestStartCheckoutReusesExistingPendingCheckout(t *testing.T) {
	repository := &fakePaymentRepository{
		reuseCheckout: &CheckoutStartResult{
			OrderID:     testOrderID,
			CheckoutURL: "https://checkout.infinitepay.com.br/existing",
			Reused:      true,
		},
	}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.StartCheckout(context.Background(), testOrderID)
	if err != nil {
		t.Fatalf("expected reused checkout, got %v", err)
	}
	if !result.Reused || result.CheckoutURL != "https://checkout.infinitepay.com.br/existing" {
		t.Fatalf("expected reused checkout result, got %#v", result)
	}
	if gateway.createCalls != 0 {
		t.Fatal("expected provider not to be called when checkout is reused")
	}
}

func TestStartCheckoutPaidOrderDoesNotCreateCheckout(t *testing.T) {
	repository := &fakePaymentRepository{checkoutErr: ErrOrderAlreadyPaid}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	_, err := service.StartCheckout(context.Background(), testOrderID)
	if !errors.Is(err, ErrOrderAlreadyPaid) {
		t.Fatalf("expected ErrOrderAlreadyPaid, got %v", err)
	}
	if gateway.createCalls != 0 {
		t.Fatal("expected provider not to be called for paid order")
	}
}

func TestConfirmReturnMarksPaidOnlyWhenPaymentCheckIsAuthoritative(t *testing.T) {
	installments := 2
	paidAt := time.Date(2026, 9, 11, 19, 30, 0, 0, time.UTC)
	repository := &fakePaymentRepository{
		target: PaymentVerificationTarget{
			OrderID:       testOrderID,
			OrderNumber:   1001,
			OrderStatus:   OrderStatusPendingPayment,
			PaymentStatus: PaymentStatusPending,
			TotalCents:    6970,
		},
	}
	gateway := &fakePaymentGateway{
		checkResult: PaymentCheckResult{
			Success:         true,
			Paid:            true,
			AmountCents:     6970,
			PaidAmountCents: 6990,
			Installments:    &installments,
			CaptureMethod:   "credit_card",
		},
	}
	service := NewService(repository, gateway, "printlab", "https://printlab.example", WithClock(func() time.Time {
		return paidAt
	}))

	result, err := service.ConfirmReturn(context.Background(), ReturnInput{
		OrderNSU:       testOrderID,
		TransactionNSU: "txn_123",
		Slug:           "slug_123",
	})
	if err != nil {
		t.Fatalf("expected confirmed payment, got %v", err)
	}
	if result.Status != ReturnStatusConfirmed || result.OrderID != testOrderID {
		t.Fatalf("expected confirmed return result, got %#v", result)
	}
	if repository.markPaidCalls != 1 {
		t.Fatalf("expected MarkPaid once, got %d", repository.markPaidCalls)
	}
	if repository.lastPayment.AmountCents != 6970 || repository.lastPayment.PaidAmountCents != 6990 {
		t.Fatalf("expected amount and paid_amount to be persisted independently, got %#v", repository.lastPayment)
	}
	if repository.lastPayment.TransactionNSU != "txn_123" || repository.lastPayment.InvoiceSlug != "slug_123" || repository.lastPayment.Installments == nil || *repository.lastPayment.Installments != 2 || repository.lastPayment.CaptureMethod != "credit_card" {
		t.Fatalf("expected provider fields to be persisted, got %#v", repository.lastPayment)
	}
	if !repository.lastPaidAt.Equal(paidAt) {
		t.Fatalf("expected paid_at %v, got %v", paidAt, repository.lastPaidAt)
	}
}

func TestConfirmReturnLeavesPendingWhenProviderSaysNotPaid(t *testing.T) {
	repository := &fakePaymentRepository{target: pendingTargetFixture()}
	gateway := &fakePaymentGateway{checkResult: PaymentCheckResult{Success: true, Paid: false}}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.ConfirmReturn(context.Background(), returnInputFixture())
	if err != nil {
		t.Fatalf("expected pending result without error, got %v", err)
	}
	if result.Status != ReturnStatusPending {
		t.Fatalf("expected pending status, got %#v", result)
	}
	if repository.markPaidCalls != 0 {
		t.Fatal("expected not to mark paid when provider says paid=false")
	}
}

func TestConfirmReturnDoesNotMarkPaidOnAmountMismatch(t *testing.T) {
	repository := &fakePaymentRepository{target: pendingTargetFixture()}
	gateway := &fakePaymentGateway{checkResult: PaymentCheckResult{Success: true, Paid: true, AmountCents: 9999, PaidAmountCents: 9999}}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.ConfirmReturn(context.Background(), returnInputFixture())
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("expected ErrAmountMismatch, got %v", err)
	}
	if result.Status != ReturnStatusUnavailable {
		t.Fatalf("expected unavailable status, got %#v", result)
	}
	if repository.markPaidCalls != 0 {
		t.Fatal("expected not to mark paid on amount mismatch")
	}
}

func TestConfirmReturnDoesNotMarkPaidWhenProviderIsUnavailable(t *testing.T) {
	repository := &fakePaymentRepository{target: pendingTargetFixture()}
	gateway := &fakePaymentGateway{checkErr: ErrProviderUnavailable}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.ConfirmReturn(context.Background(), returnInputFixture())
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if result.Status != ReturnStatusUnavailable {
		t.Fatalf("expected unavailable status, got %#v", result)
	}
	if repository.markPaidCalls != 0 {
		t.Fatal("expected not to mark paid when provider is unavailable")
	}
}

func TestConfirmReturnIsIdempotentForAlreadyPaidOrder(t *testing.T) {
	repository := &fakePaymentRepository{
		target: PaymentVerificationTarget{
			OrderID:       testOrderID,
			OrderNumber:   1001,
			OrderStatus:   OrderStatusPaid,
			PaymentStatus: PaymentStatusPaid,
			TotalCents:    6970,
		},
	}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.ConfirmReturn(context.Background(), returnInputFixture())
	if err != nil {
		t.Fatalf("expected idempotent confirmed result, got %v", err)
	}
	if result.Status != ReturnStatusConfirmed || result.OrderID != testOrderID {
		t.Fatalf("expected confirmed result, got %#v", result)
	}
	if gateway.checkCalls != 0 || repository.markPaidCalls != 0 {
		t.Fatal("expected duplicate return not to call provider or mark paid")
	}
}

func TestConfirmWebhookMarksPaidOnlyAfterPaymentCheck(t *testing.T) {
	installments := 1
	repository := &fakePaymentRepository{target: pendingTargetFixture()}
	gateway := &fakePaymentGateway{
		checkResult: PaymentCheckResult{
			Success:         true,
			Paid:            true,
			AmountCents:     6970,
			PaidAmountCents: 6970,
			Installments:    &installments,
			CaptureMethod:   "pix",
		},
	}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	webhookAmount := int64(6970)
	result, err := service.ConfirmWebhook(context.Background(), WebhookInput{
		OrderNSU:       testOrderID,
		TransactionNSU: "txn_123",
		InvoiceSlug:    "slug_123",
		AmountCents:    &webhookAmount,
	})
	if err != nil {
		t.Fatalf("expected confirmed webhook, got %v", err)
	}
	if result.Status != ReturnStatusConfirmed || result.Message != ReturnMessageVerified {
		t.Fatalf("expected verified webhook result, got %#v", result)
	}
	if repository.markPaidCalls != 1 {
		t.Fatalf("expected MarkPaid once, got %d", repository.markPaidCalls)
	}
	if gateway.lastCheckRequest.OrderNSU != testOrderID || gateway.lastCheckRequest.TransactionNSU != "txn_123" || gateway.lastCheckRequest.Slug != "slug_123" {
		t.Fatalf("expected webhook identifiers to be checked server-side, got %#v", gateway.lastCheckRequest)
	}
	if repository.lastPayment.AmountCents != 6970 || repository.lastPayment.PaidAmountCents != 6970 {
		t.Fatalf("expected payment_check amounts to be persisted, got %#v", repository.lastPayment)
	}
}

func TestConfirmWebhookRejectsPendingUnavailableAndAmountMismatch(t *testing.T) {
	tests := []struct {
		name        string
		checkResult PaymentCheckResult
		checkErr    error
		wantErr     error
		wantStatus  string
	}{
		{
			name:        "pending",
			checkResult: PaymentCheckResult{Success: true, Paid: false},
			wantErr:     ErrPaymentNotConfirmed,
			wantStatus:  ReturnStatusPending,
		},
		{
			name:        "fake webhook amount still needs payment check",
			checkResult: PaymentCheckResult{Success: false, Paid: false},
			wantErr:     ErrPaymentNotConfirmed,
			wantStatus:  ReturnStatusPending,
		},
		{
			name:        "amount mismatch",
			checkResult: PaymentCheckResult{Success: true, Paid: true, AmountCents: 9999, PaidAmountCents: 9999},
			wantErr:     ErrAmountMismatch,
			wantStatus:  ReturnStatusUnavailable,
		},
		{
			name:       "provider timeout",
			checkErr:   NewProviderError(ProviderOperationPaymentCheck, ProviderCategoryTimeout, 0, ErrProviderUnavailable),
			wantErr:    ErrProviderUnavailable,
			wantStatus: ReturnStatusUnavailable,
		},
		{
			name:       "provider 500",
			checkErr:   NewProviderError(ProviderOperationPaymentCheck, ProviderCategoryHTTP5xx, 500, ErrProviderUnavailable),
			wantErr:    ErrProviderUnavailable,
			wantStatus: ReturnStatusUnavailable,
		},
		{
			name:       "provider invalid json",
			checkErr:   NewProviderError(ProviderOperationPaymentCheck, ProviderCategoryInvalidJSON, 0, ErrProviderUnavailable),
			wantErr:    ErrProviderUnavailable,
			wantStatus: ReturnStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhookAmount := int64(6970)
			repository := &fakePaymentRepository{target: pendingTargetFixture()}
			gateway := &fakePaymentGateway{checkResult: tt.checkResult, checkErr: tt.checkErr}
			service := NewService(repository, gateway, "printlab", "https://printlab.example")

			result, err := service.ConfirmWebhook(context.Background(), WebhookInput{
				OrderNSU:       testOrderID,
				TransactionNSU: "txn_123",
				InvoiceSlug:    "slug_123",
				AmountCents:    &webhookAmount,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if result.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %#v", tt.wantStatus, result)
			}
			if repository.markPaidCalls != 0 {
				t.Fatal("expected webhook not to mark paid")
			}
		})
	}
}

func TestConfirmWebhookIsIdempotentForAlreadyPaidOrder(t *testing.T) {
	repository := &fakePaymentRepository{
		target: PaymentVerificationTarget{
			OrderID:       testOrderID,
			OrderNumber:   1001,
			OrderStatus:   OrderStatusPaid,
			PaymentStatus: PaymentStatusPaid,
			TotalCents:    6970,
		},
	}
	gateway := &fakePaymentGateway{}
	service := NewService(repository, gateway, "printlab", "https://printlab.example")

	result, err := service.ConfirmWebhook(context.Background(), webhookInputFixture())
	if err != nil {
		t.Fatalf("expected idempotent webhook result, got %v", err)
	}
	if result.Status != ReturnStatusConfirmed || result.Message != ReturnMessageAlreadyPaid {
		t.Fatalf("expected already paid result, got %#v", result)
	}
	if gateway.checkCalls != 0 || repository.markPaidCalls != 0 {
		t.Fatal("expected duplicate webhook not to call provider or mark paid")
	}
}

func TestConfirmReturnAndWebhookAreDeterministicallyIdempotent(t *testing.T) {
	t.Run("return then webhook", func(t *testing.T) {
		repository := &fakePaymentRepository{target: pendingTargetFixture()}
		gateway := &fakePaymentGateway{checkResult: PaymentCheckResult{Success: true, Paid: true, AmountCents: 6970, PaidAmountCents: 6970}}
		service := NewService(repository, gateway, "printlab", "https://printlab.example")

		if _, err := service.ConfirmReturn(context.Background(), returnInputFixture()); err != nil {
			t.Fatalf("expected return confirmation, got %v", err)
		}
		result, err := service.ConfirmWebhook(context.Background(), webhookInputFixture())
		if err != nil {
			t.Fatalf("expected duplicate webhook confirmation, got %v", err)
		}
		if result.Message != ReturnMessageAlreadyPaid || repository.markPaidCalls != 1 {
			t.Fatalf("expected webhook to be idempotent after return, result=%#v markPaidCalls=%d", result, repository.markPaidCalls)
		}
	})

	t.Run("webhook then return", func(t *testing.T) {
		repository := &fakePaymentRepository{target: pendingTargetFixture()}
		gateway := &fakePaymentGateway{checkResult: PaymentCheckResult{Success: true, Paid: true, AmountCents: 6970, PaidAmountCents: 6970}}
		service := NewService(repository, gateway, "printlab", "https://printlab.example")

		if _, err := service.ConfirmWebhook(context.Background(), webhookInputFixture()); err != nil {
			t.Fatalf("expected webhook confirmation, got %v", err)
		}
		result, err := service.ConfirmReturn(context.Background(), returnInputFixture())
		if err != nil {
			t.Fatalf("expected duplicate return confirmation, got %v", err)
		}
		if result.Message != ReturnMessageAlreadyPaid || repository.markPaidCalls != 1 {
			t.Fatalf("expected return to be idempotent after webhook, result=%#v markPaidCalls=%d", result, repository.markPaidCalls)
		}
	})
}

func checkoutOrderFixture() CheckoutOrder {
	return CheckoutOrder{
		ID:          testOrderID,
		OrderNSU:    testOrderID,
		OrderNumber: 1001,
		Status:      OrderStatusPendingPayment,
		TotalCents:  6970,
		Items: []CheckoutOrderItem{
			{
				ProductName:    "Produto Real",
				VariantName:    "Padrao",
				Quantity:       3,
				UnitPriceCents: 1990,
			},
		},
		Customer: CheckoutCustomer{
			Name:  "Joao Silva",
			Email: "joao@example.com",
			Phone: "+5527999999999",
		},
		Address: CheckoutAddress{
			PostalCode:   "29100000",
			Street:       "Rua Um",
			Number:       "12A",
			Complement:   "Apto 302",
			Neighborhood: "Centro",
		},
		Shipping: CheckoutShipping{
			ServiceName: "SEDEX",
			PriceCents:  1000,
		},
	}
}

func pendingTargetFixture() PaymentVerificationTarget {
	return PaymentVerificationTarget{
		OrderID:       testOrderID,
		OrderNumber:   1001,
		OrderStatus:   OrderStatusPendingPayment,
		PaymentStatus: PaymentStatusPending,
		TotalCents:    6970,
	}
}

func returnInputFixture() ReturnInput {
	return ReturnInput{
		OrderNSU:       testOrderID,
		TransactionNSU: "txn_123",
		Slug:           "slug_123",
	}
}

func webhookInputFixture() WebhookInput {
	return WebhookInput{
		OrderNSU:       testOrderID,
		TransactionNSU: "txn_123",
		InvoiceSlug:    "slug_123",
	}
}

type fakePaymentRepository struct {
	checkoutOrder CheckoutOrder
	reuseCheckout *CheckoutStartResult
	checkoutErr   error
	target        PaymentVerificationTarget
	targetErr     error
	markPaidErr   error

	markPaidCalls int
	lastPayment   VerifiedPayment
	lastPaidAt    time.Time
}

func (r *fakePaymentRepository) CreateOrReuseCheckout(ctx context.Context, orderID string, create CheckoutCreator) (CheckoutStartResult, error) {
	if r.checkoutErr != nil {
		return CheckoutStartResult{OrderID: orderID}, r.checkoutErr
	}
	if r.reuseCheckout != nil {
		return *r.reuseCheckout, nil
	}
	order := r.checkoutOrder
	if order.ID == "" {
		order = checkoutOrderFixture()
	}
	created, err := create(ctx, order)
	if err != nil {
		return CheckoutStartResult{OrderID: order.ID}, err
	}

	return CheckoutStartResult{OrderID: order.ID, CheckoutURL: created.URL}, nil
}

func (r *fakePaymentRepository) PaymentTargetByOrderNSU(_ context.Context, _ string) (PaymentVerificationTarget, error) {
	if r.targetErr != nil {
		return PaymentVerificationTarget{}, r.targetErr
	}
	if r.target.OrderID == "" {
		return pendingTargetFixture(), nil
	}

	return r.target, nil
}

func (r *fakePaymentRepository) MarkPaid(_ context.Context, _ string, payment VerifiedPayment, paidAt time.Time) (ReturnResult, error) {
	r.markPaidCalls++
	r.lastPayment = payment
	r.lastPaidAt = paidAt
	if r.markPaidErr != nil {
		return ReturnResult{Status: ReturnStatusUnavailable, OrderID: testOrderID}, r.markPaidErr
	}
	if r.target.OrderID == "" {
		r.target = pendingTargetFixture()
	}
	r.target.OrderStatus = OrderStatusPaid
	r.target.PaymentStatus = PaymentStatusPaid

	return ReturnResult{Status: ReturnStatusConfirmed, OrderID: testOrderID}, nil
}

type fakePaymentGateway struct {
	createResult CheckoutCreated
	createErr    error
	checkResult  PaymentCheckResult
	checkErr     error

	createCalls      int
	checkCalls       int
	lastRequest      CheckoutRequest
	lastCheckRequest PaymentCheckRequest
}

func (g *fakePaymentGateway) CreateCheckout(_ context.Context, request CheckoutRequest) (CheckoutCreated, error) {
	g.createCalls++
	g.lastRequest = request
	if g.createErr != nil {
		return CheckoutCreated{}, g.createErr
	}
	if g.createResult.URL != "" {
		return g.createResult, nil
	}

	return CheckoutCreated{URL: "https://checkout.infinitepay.com.br/checkout-slug"}, nil
}

func (g *fakePaymentGateway) CheckPayment(_ context.Context, request PaymentCheckRequest) (PaymentCheckResult, error) {
	g.checkCalls++
	g.lastCheckRequest = request
	if g.checkErr != nil {
		return PaymentCheckResult{}, g.checkErr
	}

	return g.checkResult, nil
}
