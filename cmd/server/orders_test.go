package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
)

const orderID = "22222222-2222-2222-2222-222222222222"

func TestCheckoutReviewGetWithoutCartRedirectsToCart(t *testing.T) {
	service := &fakeOrderReviewService{}
	req := httptest.NewRequest(http.MethodGet, "/checkout/revisao", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/carrinho" {
		t.Fatalf("expected redirect to /carrinho, got %q", rec.Header().Get("Location"))
	}
	if service.reviewCalls != 0 {
		t.Fatal("expected missing cart not to call order service")
	}
}

func TestCheckoutReviewGetRedirectsByPrecondition(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		location string
	}{
		{name: "empty cart", err: ordersdomain.ErrEmptyCart, location: "/carrinho"},
		{name: "unavailable items", err: ordersdomain.ErrUnavailableItems, location: "/carrinho"},
		{name: "missing details", err: ordersdomain.ErrDetailsRequired, location: "/checkout/dados"},
		{name: "missing shipping", err: ordersdomain.ErrShippingRequired, location: "/checkout/frete"},
		{name: "expired shipping", err: ordersdomain.ErrShippingExpired, location: "/checkout/frete"},
		{name: "changed shipping hash", err: ordersdomain.ErrShippingChanged, location: "/checkout/frete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeOrderReviewService{reviewErr: tt.err}
			cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
			req := httptest.NewRequest(http.MethodGet, "/checkout/revisao", nil)
			addValidCartCookie(t, cookies, req)
			rec := httptest.NewRecorder()

			newTestHandlerWithOrders(t, service, cookies, "").ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
			}
			if rec.Header().Get("Location") != tt.location {
				t.Fatalf("expected redirect to %s, got %q", tt.location, rec.Header().Get("Location"))
			}
		})
	}
}

func TestCheckoutReviewGetWithValidStateReturnsOK(t *testing.T) {
	service := &fakeOrderReviewService{reviewPage: orderReviewPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/revisao", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	for _, expected := range []string{
		"Etapa 3 - Revisao",
		"Produto Real",
		"Padrao",
		"R$ 79,80",
		"R$ 18,90",
		"R$ 98,70",
		"PAC",
		"***.***.***-25",
		`href="/checkout/dados"`,
		`href="/checkout/frete"`,
		`name="review_fingerprint"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected review page to contain %q", expected)
		}
	}
	for _, forbidden := range []string{`name="unit_price"`, `name="shipping_price"`, `name="subtotal"`, `name="total"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected review form not to include authoritative field %s", forbidden)
		}
	}
	for _, forbiddenOperationalData := range []string{
		"SKU",
		"PR-001",
		"Tempo por unidade",
		"Filamento por unidade",
		"PLA",
		"Azul",
		"Cor principal",
		"12 g",
		"Pacote",
		"Caixa Media",
	} {
		if strings.Contains(body, forbiddenOperationalData) {
			t.Fatalf("expected review page not to render operational data %q", forbiddenOperationalData)
		}
	}
}

func TestCheckoutReviewPostRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeOrderReviewService{}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := orderReviewFormRequest(url.Values{"review_fingerprint": {"fingerprint-1"}})
	req.Header.Set("Origin", "https://evil.example")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if service.confirmCalls != 0 {
		t.Fatal("expected cross-site request not to call order service")
	}
}

func TestCheckoutReviewPostStaleFingerprintRerenders(t *testing.T) {
	page := orderReviewPageFixture()
	page.Message = ordersdomain.StaleReviewMessage
	service := &fakeOrderReviewService{
		confirmResult: ordersdomain.ConfirmResult{Page: page},
		confirmErr:    ordersdomain.ErrStaleReview,
	}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := orderReviewFormRequest(url.Values{"review_fingerprint": {"old-fingerprint"}})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	if !strings.Contains(rec.Body.String(), ordersdomain.StaleReviewMessage) {
		t.Fatal("expected stale review message")
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected stale review not to expire cookie, got %#v", rec.Result().Cookies())
	}
}

func TestCheckoutReviewPostValidRedirectsAndExpiresCookie(t *testing.T) {
	service := &fakeOrderReviewService{
		confirmResult: ordersdomain.ConfirmResult{
			OrderID:      orderID,
			OrderNumber:  1001,
			Status:       ordersdomain.StatusPendingPayment,
			ExpireCookie: true,
		},
	}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := orderReviewFormRequest(url.Values{"review_fingerprint": {"fingerprint-1"}})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/pedido/"+orderID {
		t.Fatalf("expected order redirect, got %q", rec.Header().Get("Location"))
	}
	resultCookies := rec.Result().Cookies()
	if len(resultCookies) != 1 || resultCookies[0].Name != cartdomain.CookieName || resultCookies[0].MaxAge != -1 {
		t.Fatalf("expected expired cart cookie, got %#v", resultCookies)
	}
	if service.lastFingerprint != "fingerprint-1" {
		t.Fatalf("expected submitted fingerprint only, got %q", service.lastFingerprint)
	}
}

func TestOrderPageWithValidOrderReturnsOK(t *testing.T) {
	service := &fakeOrderReviewService{orderPage: orderPageFixture()}
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodGet, "/pedido/"+orderID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, service, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	for _, expected := range []string{"Pedido #1001", "Aguardando pagamento", "Pagar agora", "ambiente seguro da InfinitePay", "Produto Real", "Padrao", "Quantidade", "2", "PAC", "R$ 98,70"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected order page to contain %q", expected)
		}
	}
	for _, forbiddenPII := range []string{"52998224725", "***.***.***-25", "joao@example.com", "+5527999999999", "Rua Um", "29100-000"} {
		if strings.Contains(body, forbiddenPII) {
			t.Fatalf("expected order page not to render PII %q", forbiddenPII)
		}
	}
	for _, forbiddenOperationalData := range []string{
		"SKU",
		"PR-001",
		"Tempo por unidade",
		"Filamento por unidade",
		"PLA",
		"Azul",
		"Cor principal",
		"12 g",
		"Pacote",
		"Caixa Media",
	} {
		if strings.Contains(body, forbiddenOperationalData) {
			t.Fatalf("expected order page not to render operational data %q", forbiddenOperationalData)
		}
	}
	for _, forbiddenPaymentData := range []string{"checkout.infinitepay.com.br", "transaction_nsu", "invoice_slug"} {
		if strings.Contains(body, forbiddenPaymentData) {
			t.Fatalf("expected order page not to render payment technical data %q", forbiddenPaymentData)
		}
	}
}

func TestOrderPageWithoutPaymentConfigShowsSafeUnavailableState(t *testing.T) {
	service := &fakeOrderReviewService{orderPage: orderPageFixture()}
	req := httptest.NewRequest(http.MethodGet, "/pedido/"+orderID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Pagamento temporariamente indisponivel") {
		t.Fatal("expected unavailable payment message")
	}
	if strings.Contains(body, "Pagar agora") {
		t.Fatal("expected payment button not to render when service is unavailable")
	}
}

func TestOrderPagePaidShowsConfirmedState(t *testing.T) {
	page := orderPageFixture()
	page.Status = ordersdomain.StatusPaid
	page.StatusLabel = "Pagamento confirmado"
	service := &fakeOrderReviewService{orderPage: page}
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodGet, "/pedido/"+orderID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, service, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Pagamento confirmado") {
		t.Fatal("expected paid status on order page")
	}
	if strings.Contains(body, "Pagar agora") {
		t.Fatal("expected paid order not to render pay button")
	}
}

func TestOrderPageInvalidUUIDReturnsNotFound(t *testing.T) {
	service := &fakeOrderReviewService{}
	req := httptest.NewRequest(http.MethodGet, "/pedido/1001", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if service.getCalls != 0 {
		t.Fatal("expected invalid UUID not to call order service")
	}
}

func TestOrderPageMissingOrderReturnsNotFound(t *testing.T) {
	service := &fakeOrderReviewService{getErr: ordersdomain.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/pedido/"+orderID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestStartPaymentRejectsCrossSiteOrigin(t *testing.T) {
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/pedido/"+orderID+"/pagar", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if payment.startCalls != 0 {
		t.Fatal("expected cross-site payment request not to call service")
	}
}

func TestStartPaymentInvalidOrderIDReturnsNotFound(t *testing.T) {
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodPost, "/pedido/1001/pagar", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if payment.startCalls != 0 {
		t.Fatal("expected invalid order id not to call service")
	}
}

func TestStartPaymentValidOrderRedirectsToInfinitePay(t *testing.T) {
	payment := &fakePaymentService{
		available: true,
		startResult: paymentsdomain.CheckoutStartResult{
			OrderID:     orderID,
			CheckoutURL: "https://checkout.infinitepay.com.br/checkout-slug",
		},
	}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/pedido/"+orderID+"/pagar", nil)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "https://checkout.infinitepay.com.br/checkout-slug" {
		t.Fatalf("expected InfinitePay redirect, got %q", rec.Header().Get("Location"))
	}
	if payment.lastStartOrderID != orderID {
		t.Fatalf("expected order ID passed to payment service, got %q", payment.lastStartOrderID)
	}
}

func TestStartPaymentPaidOrderRedirectsBackToOrder(t *testing.T) {
	payment := &fakePaymentService{
		available:   true,
		startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID},
		startErr:    paymentsdomain.ErrOrderAlreadyPaid,
	}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/pedido/"+orderID+"/pagar", nil)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/pedido/"+orderID {
		t.Fatalf("expected order redirect, got %q", rec.Header().Get("Location"))
	}
}

func TestPaymentReturnMissingParamsShowsSafeError(t *testing.T) {
	payment := &fakePaymentService{available: true, returnErr: paymentsdomain.ErrInvalidReturn}
	req := httptest.NewRequest(http.MethodGet, "/pagamento/retorno", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Pagamento nao confirmado") {
		t.Fatal("expected safe payment error page")
	}
	if strings.Contains(rec.Body.String(), "order_nsu") {
		t.Fatal("expected return page not to reflect query parameter names or raw values")
	}
}

func TestPaymentReturnUnknownOrderShowsNotFound(t *testing.T) {
	payment := &fakePaymentService{available: true, returnErr: paymentsdomain.ErrPaymentNotFound}
	req := paymentReturnRequest()
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestPaymentReturnPendingShowsPendingMessage(t *testing.T) {
	payment := &fakePaymentService{
		available:    true,
		returnResult: paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusPending, OrderID: orderID},
	}
	req := paymentReturnRequest()
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Pagamento ainda nao foi confirmado") {
		t.Fatal("expected pending payment message")
	}
	if payment.lastReturnInput.OrderNSU != orderID || payment.lastReturnInput.TransactionNSU != "txn_123" || payment.lastReturnInput.Slug != "slug_123" {
		t.Fatalf("expected documented return params only, got %#v", payment.lastReturnInput)
	}
}

func TestPaymentReturnConfirmedRedirectsToOrder(t *testing.T) {
	payment := &fakePaymentService{
		available:    true,
		returnResult: paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusConfirmed, OrderID: orderID},
	}
	req := paymentReturnRequest()
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/pedido/"+orderID+"?pagamento=confirmado" {
		t.Fatalf("expected confirmed order redirect, got %q", rec.Header().Get("Location"))
	}
}

func orderReviewFormRequest(values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/revisao", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func paymentReturnRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/pagamento/retorno?order_nsu="+orderID+"&transaction_nsu=txn_123&slug=slug_123&capture_method=ignored&receipt_url=https://example.test", nil)
}

func orderReviewPageFixture() ordersdomain.ReviewPage {
	printTime := 90
	filamentWeight := int64(12000)
	deliveryDays := 5
	return ordersdomain.ReviewPage{
		Items: []ordersdomain.ReviewItem{
			{
				CartItemID:                       "item-1",
				ProductID:                        "33333333-3333-3333-3333-333333333333",
				VariantID:                        "44444444-4444-4444-4444-444444444444",
				ProductName:                      "Produto Real",
				ProductSlug:                      "produto-real",
				VariantName:                      "Padrao",
				VariantSlug:                      "padrao",
				SKU:                              "PR-001",
				HasVariant:                       true,
				Quantity:                         2,
				UnitPriceCents:                   3990,
				UnitPriceBRL:                     "R$ 39,90",
				LineTotalCents:                   7980,
				LineTotalBRL:                     "R$ 79,80",
				UnitPrintTimeMinutes:             &printTime,
				UnitEstimatedFilamentWeightMg:    &filamentWeight,
				UnitEstimatedFilamentWeightLabel: "12 g",
				Filaments: []ordersdomain.ReviewItemFilament{
					{
						MaterialName:             "PLA",
						MaterialSlug:             "pla",
						ColorName:                "Azul",
						ColorSlug:                "azul",
						HexColor:                 "#1187F4",
						EstimatedWeightMgPerUnit: 12000,
						EstimatedWeightLabel:     "12 g",
						Label:                    "Cor principal",
					},
				},
			},
		},
		Customer: ordersdomain.ReviewCustomer{
			FullName:  "Joao Silva",
			Email:     "joao@example.com",
			Phone:     "+5527999999999",
			CPF:       "52998224725",
			MaskedCPF: "***.***.***-25",
		},
		Address: ordersdomain.ReviewAddress{
			PostalCode:  "29100000",
			Street:      "Rua Um",
			Number:      "12A",
			Complement:  "Apto 302",
			District:    "Centro",
			City:        "Vila Velha",
			State:       "ES",
			CountryCode: "BR",
			LineOne:     "Rua Um, 12A - Apto 302",
			LineTwo:     "Centro - Vila Velha/ES - CEP 29100-000",
		},
		Shipping: ordersdomain.ReviewShipping{
			Provider:         "superfrete",
			ServiceCode:      "1",
			ServiceName:      "PAC",
			CarrierName:      "Correios",
			DeliveryTimeDays: &deliveryDays,
			DeliveryTime:     "5 dias uteis",
			ShippingBoxName:  "Caixa Media",
			PriceCents:       1890,
			PriceBRL:         "R$ 18,90",
			PackageWeightG:   720,
			PackageHeightMM:  120,
			PackageWidthMM:   160,
			PackageLengthMM:  240,
		},
		ProductsSubtotalCents: 7980,
		ProductsSubtotalBRL:   "R$ 79,80",
		ShippingPriceCents:    1890,
		ShippingPriceBRL:      "R$ 18,90",
		TotalCents:            9870,
		TotalBRL:              "R$ 98,70",
		Fingerprint:           "fingerprint-1",
	}
}

func orderPageFixture() ordersdomain.OrderPage {
	review := orderReviewPageFixture()
	return ordersdomain.OrderPage{
		ID:                  orderID,
		OrderNumber:         1001,
		OrderNumberLabel:    "#1001",
		Status:              ordersdomain.StatusPendingPayment,
		StatusLabel:         "Aguardando pagamento",
		CreatedAt:           time.Now(),
		ProductsSubtotalBRL: "R$ 79,80",
		ShippingPriceBRL:    "R$ 18,90",
		TotalBRL:            "R$ 98,70",
		Items: []ordersdomain.OrderItem{
			{
				ProductName:                      review.Items[0].ProductName,
				VariantName:                      review.Items[0].VariantName,
				SKU:                              review.Items[0].SKU,
				HasVariant:                       true,
				Quantity:                         review.Items[0].Quantity,
				UnitPriceBRL:                     review.Items[0].UnitPriceBRL,
				LineTotalBRL:                     review.Items[0].LineTotalBRL,
				UnitPrintTimeMinutes:             review.Items[0].UnitPrintTimeMinutes,
				UnitEstimatedFilamentWeightMg:    review.Items[0].UnitEstimatedFilamentWeightMg,
				UnitEstimatedFilamentWeightLabel: review.Items[0].UnitEstimatedFilamentWeightLabel,
				Filaments:                        review.Items[0].Filaments,
			},
		},
		Shipping: ordersdomain.OrderShipping{
			ServiceName:      review.Shipping.ServiceName,
			CarrierName:      review.Shipping.CarrierName,
			DeliveryTimeDays: review.Shipping.DeliveryTimeDays,
			DeliveryTime:     review.Shipping.DeliveryTime,
			ShippingBoxName:  review.Shipping.ShippingBoxName,
			PriceBRL:         review.Shipping.PriceBRL,
			PackageWeightG:   review.Shipping.PackageWeightG,
			PackageHeightMM:  review.Shipping.PackageHeightMM,
			PackageWidthMM:   review.Shipping.PackageWidthMM,
			PackageLengthMM:  review.Shipping.PackageLengthMM,
		},
	}
}

func newTestHandlerWithOrders(t *testing.T, service orderReviewService, cookies *cartdomain.CookieManager, siteURL string) http.Handler {
	t.Helper()

	return newTestHandlerWithOrdersAndPayment(t, service, nil, cookies, siteURL)
}

func newTestHandlerWithOrdersAndPayment(t *testing.T, service orderReviewService, payment paymentService, cookies *cartdomain.CookieManager, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServicesAndOrders(db, nil, nil, nil, nil, service, cookies, nil, payment, siteURL)
}

type fakeOrderReviewService struct {
	reviewPage    ordersdomain.ReviewPage
	confirmResult ordersdomain.ConfirmResult
	orderPage     ordersdomain.OrderPage
	reviewErr     error
	confirmErr    error
	getErr        error

	reviewCalls  int
	confirmCalls int
	getCalls     int

	lastStale       bool
	lastFingerprint string
	lastOrderID     string
}

func (s *fakeOrderReviewService) Review(_ context.Context, _ []byte, stale bool) (ordersdomain.ReviewPage, error) {
	s.reviewCalls++
	s.lastStale = stale
	if s.reviewErr != nil {
		return ordersdomain.ReviewPage{}, s.reviewErr
	}

	return s.reviewPage, nil
}

func (s *fakeOrderReviewService) Confirm(_ context.Context, _ []byte, expectedFingerprint string) (ordersdomain.ConfirmResult, error) {
	s.confirmCalls++
	s.lastFingerprint = expectedFingerprint
	if s.confirmErr != nil {
		return s.confirmResult, s.confirmErr
	}

	return s.confirmResult, nil
}

func (s *fakeOrderReviewService) Get(_ context.Context, orderID string) (ordersdomain.OrderPage, error) {
	s.getCalls++
	s.lastOrderID = orderID
	if s.getErr != nil {
		return ordersdomain.OrderPage{}, s.getErr
	}

	return s.orderPage, nil
}

var _ orderReviewService = (*fakeOrderReviewService)(nil)

type fakePaymentService struct {
	available    bool
	startResult  paymentsdomain.CheckoutStartResult
	returnResult paymentsdomain.ReturnResult
	startErr     error
	returnErr    error

	startCalls       int
	returnCalls      int
	lastStartOrderID string
	lastReturnInput  paymentsdomain.ReturnInput
}

func (s *fakePaymentService) Available() bool {
	return s.available
}

func (s *fakePaymentService) StartCheckout(_ context.Context, orderID string) (paymentsdomain.CheckoutStartResult, error) {
	s.startCalls++
	s.lastStartOrderID = orderID
	if s.startErr != nil {
		return s.startResult, s.startErr
	}
	if s.startResult.CheckoutURL == "" {
		return paymentsdomain.CheckoutStartResult{
			OrderID:     orderID,
			CheckoutURL: "https://checkout.infinitepay.com.br/checkout-slug",
		}, nil
	}

	return s.startResult, nil
}

func (s *fakePaymentService) ConfirmReturn(_ context.Context, input paymentsdomain.ReturnInput) (paymentsdomain.ReturnResult, error) {
	s.returnCalls++
	s.lastReturnInput = input
	if s.returnErr != nil {
		return s.returnResult, s.returnErr
	}
	if s.returnResult.Status == "" {
		return paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusPending, OrderID: orderID}, nil
	}

	return s.returnResult, nil
}

var _ paymentService = (*fakePaymentService)(nil)
