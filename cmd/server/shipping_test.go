package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
)

func TestCheckoutShippingGetWithoutCartRedirectsToCart(t *testing.T) {
	service := &fakeCheckoutShippingService{}
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/carrinho" {
		t.Fatalf("expected redirect to /carrinho, got %q", rec.Header().Get("Location"))
	}
	if service.pageCalls != 0 {
		t.Fatal("expected missing cart not to call shipping service")
	}
}

func TestCheckoutShippingGetWithoutDetailsRedirectsToCheckoutDetails(t *testing.T) {
	service := &fakeCheckoutShippingService{pageErr: shipping.ErrDetailsRequired}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/checkout/dados" {
		t.Fatalf("expected redirect to /checkout/dados, got %q", rec.Header().Get("Location"))
	}
}

func TestCheckoutShippingGetShowsMissingProfileState(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.Unavailable = true
	page.Message = "Frete temporariamente indisponivel para este carrinho."
	service := &fakeCheckoutShippingService{page: page}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Frete temporariamente indisponivel para este carrinho.") ||
		!strings.Contains(body, "Frete indisponivel") {
		t.Fatal("expected missing profile state")
	}
}

func TestCheckoutShippingGetShowsNoBoxState(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.Unavailable = true
	page.Message = "Nao conseguimos calcular automaticamente o frete para este carrinho."
	service := &fakeCheckoutShippingService{page: page}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Nao conseguimos calcular automaticamente o frete para este carrinho.") {
		t.Fatal("expected no-box state")
	}
}

func TestCheckoutShippingGetWithValidQuoteReturnsOK(t *testing.T) {
	service := &fakeCheckoutShippingService{page: checkoutShippingPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Etapa 2 - Frete", "PAC", "R$ 18,90", "5 dias uteis"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected shipping page to contain %q", expected)
		}
	}
	for _, forbidden := range []string{`name="price"`, `name="package_weight_g"`, `box-medium`, `Caixa Media`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected shipping page not to expose %q", forbidden)
		}
	}
}

func TestCheckoutShippingPostRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeCheckoutShippingService{}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutShippingFormRequest(url.Values{"service_code": {"1"}})
	req.Header.Set("Origin", "https://evil.example")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if service.selectCalls != 0 {
		t.Fatal("expected cross-site request not to call shipping service")
	}
}

func TestCheckoutShippingPostValidServiceRedirects(t *testing.T) {
	service := &fakeCheckoutShippingService{selectResult: shipping.SelectResult{Page: checkoutShippingPageFixture()}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutShippingFormRequest(url.Values{
		"service_code":      {"1"},
		"price":             {"1"},
		"package_weight_g":  {"1"},
		"package_height_mm": {"1"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/checkout/frete?selecionado=1" {
		t.Fatalf("expected selected redirect, got %q", rec.Header().Get("Location"))
	}
	if service.lastServiceCode != "1" {
		t.Fatalf("expected only service_code to be submitted to service, got %q", service.lastServiceCode)
	}
}

func TestCheckoutShippingPostInvalidServiceRerenders(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.Message = "A opcao selecionada nao esta mais disponivel. Escolha uma cotacao atual."
	service := &fakeCheckoutShippingService{
		selectResult: shipping.SelectResult{Page: page},
		selectErr:    shipping.ErrInvalidService,
	}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutShippingFormRequest(url.Values{"service_code": {"33"}})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	if !strings.Contains(rec.Body.String(), "A opcao selecionada nao esta mais disponivel.") {
		t.Fatal("expected invalid selection message")
	}
}

func checkoutShippingFormRequest(values url.Values) *http.Request {
	body := strings.NewReader(values.Encode())
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/frete", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func checkoutShippingPageFixture() shipping.CheckoutShippingPage {
	return shipping.CheckoutShippingPage{
		Cart: cartdomain.CartView{
			Lines: []cartdomain.CartLine{
				{
					ID:          "item-1",
					ProductName: "Produto Real",
					VariantName: "Padrao",
					HasVariant:  true,
					Quantity:    2,
					SubtotalBRL: "R$ 79,80",
					Available:   true,
				},
			},
			SubtotalCents: 7980,
			SubtotalBRL:   "R$ 79,80",
		},
		ProductsSubtotalBRL: "R$ 79,80",
		Quotes: []shipping.ShippingQuote{
			{
				Provider:     shipping.ProviderSuperFrete,
				ServiceCode:  "1",
				ServiceName:  "PAC",
				CarrierName:  "Correios",
				PriceCents:   1890,
				PriceBRL:     "R$ 18,90",
				DeliveryTime: "5 dias uteis",
			},
		},
	}
}

func newTestHandlerWithShipping(t *testing.T, service checkoutShippingService, cookies *cartdomain.CookieManager, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServices(db, nil, nil, nil, service, cookies, nil, siteURL)
}

type fakeCheckoutShippingService struct {
	page         shipping.CheckoutShippingPage
	selectResult shipping.SelectResult
	pageErr      error
	selectErr    error

	pageCalls   int
	selectCalls int

	lastSelected    bool
	lastServiceCode string
}

func (s *fakeCheckoutShippingService) Page(_ context.Context, _ []byte, selected bool) (shipping.CheckoutShippingPage, error) {
	s.pageCalls++
	s.lastSelected = selected
	if s.pageErr != nil {
		return shipping.CheckoutShippingPage{}, s.pageErr
	}

	return s.page, nil
}

func (s *fakeCheckoutShippingService) Select(_ context.Context, _ []byte, serviceCode string) (shipping.SelectResult, error) {
	s.selectCalls++
	s.lastServiceCode = serviceCode
	if s.selectErr != nil {
		return s.selectResult, s.selectErr
	}

	return s.selectResult, nil
}

var _ checkoutShippingService = (*fakeCheckoutShippingService)(nil)
