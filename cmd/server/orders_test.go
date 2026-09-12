package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
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
const trackingID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

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
	for _, expected := range []string{"Pedido #1001", "Aguardando pagamento", "Pagar agora", "ambiente seguro da InfinitePay", "Produto Real", "Padrao", "Quantidade", "2", "PAC", "R$ 98,70", `href="/acompanhar/` + trackingID + `"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected order page to contain %q", expected)
		}
	}
	if strings.Contains(body, `href="/acompanhar/`+orderID+`"`) {
		t.Fatal("expected order tracking link not to use internal order ID")
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

func TestOrderTrackingPageWithValidIDReturnsOK(t *testing.T) {
	service := &fakeOrderReviewService{trackingPage: trackingPageFixture()}
	req := httptest.NewRequest(http.MethodGet, "/acompanhar/"+trackingID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	assertOrderTrackingHeaders(t, rec)
	body := rec.Body.String()
	for _, expected := range []string{
		`<meta name="robots" content="noindex, nofollow, noarchive">`,
		"Pedido #1004",
		"Data do pedido",
		"12/09/2026 14:30",
		"Pagamento",
		"Aguardando pagamento",
		"Producao",
		"Sera iniciada apos a confirmacao do pagamento",
		"Envio",
		"Sera preparado apos a producao",
		"Correios - PAC",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected tracking page to contain %q", expected)
		}
	}
	for _, leaked := range []string{
		orderID,
		"52998224725",
		"***.***.***-25",
		"joao@example.com",
		"+5527999999999",
		"Rua Um",
		"29100-000",
		"source_cart_id",
		"transaction_nsu",
		"invoice_slug",
		"checkout_url",
		"720",
		"120",
		"160",
		"240",
		"filamento",
		"filament",
		"PLA",
		"Azul",
		"Material",
		"Cor principal",
	} {
		if strings.Contains(body, leaked) {
			t.Fatalf("expected tracking page not to contain %q", leaked)
		}
	}
	if service.lastTrackingID != trackingID {
		t.Fatalf("expected tracking id passed to service, got %q", service.lastTrackingID)
	}
}

func TestOrderTrackingInvalidUUIDReturnsNotFound(t *testing.T) {
	service := &fakeOrderReviewService{}
	req := httptest.NewRequest(http.MethodGet, "/acompanhar/1004", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	assertOrderTrackingHeaders(t, rec)
	if service.trackCalls != 0 {
		t.Fatal("expected invalid tracking ID not to call order service")
	}
}

func TestOrderTrackingUnknownIDReturnsNotFound(t *testing.T) {
	service := &fakeOrderReviewService{trackErr: ordersdomain.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/acompanhar/"+trackingID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	assertOrderTrackingHeaders(t, rec)
	if service.trackCalls != 1 {
		t.Fatalf("expected one tracking lookup, got %d", service.trackCalls)
	}
}

func TestOrderTrackingPostMethodNotAllowed(t *testing.T) {
	service := &fakeOrderReviewService{}
	req := httptest.NewRequest(http.MethodPost, "/acompanhar/"+trackingID, nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrders(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("expected Allow GET, got %q", rec.Header().Get("Allow"))
	}
	if service.trackCalls != 0 {
		t.Fatal("expected POST not to call tracking service")
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

func TestStartPaymentProviderErrorLogsSafeDiagnostics(t *testing.T) {
	providerErr := paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationCreateCheckout, paymentsdomain.ProviderCategoryHTTP422, http.StatusUnprocessableEntity, paymentsdomain.ErrProviderUnavailable)
	providerErr.Message = "invalid customer Joao Silva joao@example.com +5527999999999 Rua Um https://checkout.infinitepay.com.br/checkout-slug"
	providerErr.Code = "invalid_handle"
	payment := &fakePaymentService{
		available:   true,
		startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID},
		startErr:    providerErr,
	}
	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})

	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/pedido/"+orderID+"/pagar", nil)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/pedido/"+orderID+"?pagamento=indisponivel" {
		t.Fatalf("expected unavailable redirect, got %q", rec.Header().Get("Location"))
	}
	logText := logs.String()
	for _, expected := range []string{"payment checkout unavailable", "provider=infinitepay", "operation=create_checkout", "status=422", "category=http_422"} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
	assertPaymentLogDoesNotLeakSensitiveData(t, logText)
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

func TestPaymentReturnProviderErrorLogsSafeDiagnostics(t *testing.T) {
	providerErr := paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationPaymentCheck, paymentsdomain.ProviderCategoryHTTP5xx, http.StatusServiceUnavailable, paymentsdomain.ErrProviderUnavailable)
	providerErr.Message = "payment check failed for txn_123 and joao@example.com"
	payment := &fakePaymentService{
		available:    true,
		returnResult: paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
		returnErr:    providerErr,
	}
	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})

	req := paymentReturnRequest()
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	logText := logs.String()
	for _, expected := range []string{"payment confirmation unavailable", "provider=infinitepay", "operation=payment_check", "status=503", "category=http_5xx"} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
	assertPaymentLogDoesNotLeakSensitiveData(t, logText)
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

func TestPaymentWebhookValidJSONReturnsOK(t *testing.T) {
	payment := &fakePaymentService{
		available:     true,
		webhookResult: paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusConfirmed, OrderID: orderID, Message: paymentsdomain.ReturnMessageVerified},
	}
	req := paymentWebhookRequest(validInfinitePayWebhookJSON())
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	assertPaymentWebhookResponse(t, rec, true, nil)
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache header, got %q", rec.Header().Get("Cache-Control"))
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON response, got %q", rec.Header().Get("Content-Type"))
	}
	if payment.webhookCalls != 1 {
		t.Fatalf("expected webhook confirmation once, got %d", payment.webhookCalls)
	}
	if payment.lastWebhookInput.OrderNSU != orderID || payment.lastWebhookInput.TransactionNSU != "txn_123" || payment.lastWebhookInput.InvoiceSlug != "slug_123" {
		t.Fatalf("expected webhook identifiers only, got %#v", payment.lastWebhookInput)
	}
}

func TestPaymentWebhookGetMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/webhooks/infinitepay", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, &fakePaymentService{available: true}, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestPaymentWebhookRejectsInvalidJSONAndOversizedBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "oversized body", body: strings.Repeat(" ", maxInfinitePayWebhookBodyBytes+1), wantStatus: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &fakePaymentService{available: true}
			req := paymentWebhookRequest(tt.body)
			rec := httptest.NewRecorder()

			newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if payment.webhookCalls != 0 {
				t.Fatal("expected invalid webhook body not to call payment service")
			}
			assertPaymentWebhookResponse(t, rec, false, stringPtr("Payload invalido"))
		})
	}
}

func TestPaymentWebhookRejectsUnsafeInputsAndMissingOrder(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		err         error
		wantMessage string
	}{
		{name: "invalid order_nsu", body: `{"invoice_slug":"slug_123","transaction_nsu":"txn_123","order_nsu":"not-a-uuid"}`, err: paymentsdomain.ErrInvalidReturn, wantMessage: "Payload invalido"},
		{name: "missing transaction_nsu", body: `{"invoice_slug":"slug_123","order_nsu":"` + orderID + `"}`, err: paymentsdomain.ErrInvalidReturn, wantMessage: "Payload invalido"},
		{name: "missing invoice_slug", body: `{"transaction_nsu":"txn_123","order_nsu":"` + orderID + `"}`, err: paymentsdomain.ErrInvalidReturn, wantMessage: "Payload invalido"},
		{name: "missing order", body: validInfinitePayWebhookJSON(), err: paymentsdomain.ErrPaymentNotFound, wantMessage: "Pedido nao encontrado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &fakePaymentService{
				available:  true,
				webhookErr: tt.err,
			}
			req := paymentWebhookRequest(tt.body)
			rec := httptest.NewRecorder()

			newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
			if payment.webhookCalls != 1 {
				t.Fatalf("expected payment service call, got %d", payment.webhookCalls)
			}
			assertPaymentWebhookResponse(t, rec, false, stringPtr(tt.wantMessage))
		})
	}
}

func TestPaymentWebhookReturnsRetryableFailureWhenPaymentCheckDoesNotConfirm(t *testing.T) {
	tests := []struct {
		name        string
		result      paymentsdomain.ReturnResult
		err         error
		wantMessage string
	}{
		{
			name:        "pending",
			result:      paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusPending, OrderID: orderID},
			err:         paymentsdomain.ErrPaymentNotConfirmed,
			wantMessage: "Pagamento ainda nao confirmado",
		},
		{
			name:        "amount mismatch",
			result:      paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
			err:         paymentsdomain.ErrAmountMismatch,
			wantMessage: "Nao foi possivel confirmar o pagamento agora",
		},
		{
			name:        "provider timeout",
			result:      paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
			err:         paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationPaymentCheck, paymentsdomain.ProviderCategoryTimeout, 0, paymentsdomain.ErrProviderUnavailable),
			wantMessage: "Nao foi possivel confirmar o pagamento agora",
		},
		{
			name:        "provider 500",
			result:      paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
			err:         paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationPaymentCheck, paymentsdomain.ProviderCategoryHTTP5xx, http.StatusInternalServerError, paymentsdomain.ErrProviderUnavailable),
			wantMessage: "Nao foi possivel confirmar o pagamento agora",
		},
		{
			name:        "provider invalid json",
			result:      paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
			err:         paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationPaymentCheck, paymentsdomain.ProviderCategoryInvalidJSON, 0, paymentsdomain.ErrProviderUnavailable),
			wantMessage: "Nao foi possivel confirmar o pagamento agora",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &fakePaymentService{
				available:     true,
				webhookResult: tt.result,
				webhookErr:    tt.err,
			}
			req := paymentWebhookRequest(validInfinitePayWebhookJSON())
			rec := httptest.NewRecorder()

			newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
			assertPaymentWebhookResponse(t, rec, false, stringPtr(tt.wantMessage))
		})
	}
}

func TestPaymentWebhookProviderErrorLogsSafeDiagnostics(t *testing.T) {
	providerErr := paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationPaymentCheck, paymentsdomain.ProviderCategoryHTTP5xx, http.StatusServiceUnavailable, paymentsdomain.ErrProviderUnavailable)
	providerErr.Message = "payment check failed for txn_123 Joao Silva joao@example.com +5527999999999 Rua Um https://checkout.infinitepay.com.br/checkout-slug"
	payment := &fakePaymentService{
		available:     true,
		webhookResult: paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable, OrderID: orderID},
		webhookErr:    providerErr,
	}
	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})

	req := paymentWebhookRequest(validInfinitePayWebhookJSON())
	rec := httptest.NewRecorder()

	newTestHandlerWithOrdersAndPayment(t, &fakeOrderReviewService{}, payment, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	logText := logs.String()
	for _, expected := range []string{"payment webhook received", "payment webhook unavailable", "provider=infinitepay", "operation=payment_check", "status=503", "category=http_5xx"} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
	assertPaymentLogDoesNotLeakSensitiveData(t, logText)
}

func orderReviewFormRequest(values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/revisao", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func paymentReturnRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/pagamento/retorno?order_nsu="+orderID+"&transaction_nsu=txn_123&slug=slug_123&capture_method=ignored&receipt_url=https://example.test", nil)
}

func paymentWebhookRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/infinitepay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func validInfinitePayWebhookJSON() string {
	return `{"invoice_slug":"slug_123","transaction_nsu":"txn_123","order_nsu":"` + orderID + `","amount":9870,"paid_amount":9870,"installments":1,"capture_method":"pix","receipt_url":"https://example.test/receipt","items":[{"name":"ignored"}]}`
}

func assertPaymentWebhookResponse(t *testing.T, rec *httptest.ResponseRecorder, success bool, message *string) {
	t.Helper()

	var response paymentWebhookResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected webhook JSON response, got %v body=%q", err, rec.Body.String())
	}
	if response.Success != success {
		t.Fatalf("expected success=%v, got %#v", success, response)
	}
	if message == nil {
		if response.Message != nil {
			t.Fatalf("expected null message, got %#v", response)
		}
		return
	}
	if response.Message == nil || *response.Message != *message {
		t.Fatalf("expected message %q, got %#v", *message, response)
	}
}

func stringPtr(value string) *string {
	return &value
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
		PublicTrackingID:    trackingID,
		TrackingURL:         "/acompanhar/" + trackingID,
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

func trackingPageFixture() ordersdomain.TrackingPage {
	return ordersdomain.TrackingPageFromRecord(ordersdomain.TrackingRecord{
		OrderNumber:      1004,
		Status:           ordersdomain.StatusPendingPayment,
		CreatedAt:        time.Date(2026, 9, 12, 14, 30, 0, 0, time.UTC),
		ProductionStatus: ordersdomain.ProductionStatusWaiting,
		ShippingStatus:   ordersdomain.ShippingStatusWaiting,
		ServiceName:      "PAC",
		CarrierName:      "Correios",
	})
}

func assertOrderTrackingHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	if robots := rec.Header().Get("X-Robots-Tag"); !strings.Contains(robots, "noindex") || !strings.Contains(robots, "nofollow") || !strings.Contains(robots, "noarchive") {
		t.Fatalf("expected noindex robots header, got %q", robots)
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected no-referrer policy, got %q", rec.Header().Get("Referrer-Policy"))
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

	return newHandlerWithServicesAndOrders(db, nil, nil, nil, nil, service, cookies, nil, payment, nil, siteURL)
}

type fakeOrderReviewService struct {
	reviewPage    ordersdomain.ReviewPage
	confirmResult ordersdomain.ConfirmResult
	orderPage     ordersdomain.OrderPage
	trackingPage  ordersdomain.TrackingPage
	reviewErr     error
	confirmErr    error
	getErr        error
	trackErr      error

	reviewCalls  int
	confirmCalls int
	getCalls     int
	trackCalls   int

	lastStale       bool
	lastFingerprint string
	lastOrderID     string
	lastTrackingID  string
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

func (s *fakeOrderReviewService) Track(_ context.Context, trackingID string) (ordersdomain.TrackingPage, error) {
	s.trackCalls++
	s.lastTrackingID = trackingID
	if s.trackErr != nil {
		return ordersdomain.TrackingPage{}, s.trackErr
	}

	return s.trackingPage, nil
}

var _ orderReviewService = (*fakeOrderReviewService)(nil)

type fakePaymentService struct {
	available     bool
	startResult   paymentsdomain.CheckoutStartResult
	returnResult  paymentsdomain.ReturnResult
	webhookResult paymentsdomain.ReturnResult
	startErr      error
	returnErr     error
	webhookErr    error

	startCalls       int
	returnCalls      int
	webhookCalls     int
	lastStartOrderID string
	lastReturnInput  paymentsdomain.ReturnInput
	lastWebhookInput paymentsdomain.WebhookInput
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

func (s *fakePaymentService) ConfirmWebhook(_ context.Context, input paymentsdomain.WebhookInput) (paymentsdomain.ReturnResult, error) {
	s.webhookCalls++
	s.lastWebhookInput = input
	if s.webhookErr != nil {
		return s.webhookResult, s.webhookErr
	}
	if s.webhookResult.Status == "" {
		return paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusPending, OrderID: orderID}, nil
	}

	return s.webhookResult, nil
}

var _ paymentService = (*fakePaymentService)(nil)

func assertPaymentLogDoesNotLeakSensitiveData(t *testing.T, logText string) {
	t.Helper()
	for _, leaked := range []string{
		"Joao",
		"joao@example.com",
		"+5527999999999",
		"Rua Um",
		"https://checkout.infinitepay.com.br/checkout-slug",
		"txn_123",
	} {
		if strings.Contains(logText, leaked) {
			t.Fatalf("expected payment log not to contain %q, got %q", leaked, logText)
		}
	}
}
