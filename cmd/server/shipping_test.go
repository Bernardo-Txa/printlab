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
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/database"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
)

func TestCheckoutShippingGetWithoutCartRedirectsToCart(t *testing.T) {
	service := &fakeCheckoutShippingService{}
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
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
	page.Message = "Frete temporariamente indisponível para este carrinho."
	service := &fakeCheckoutShippingService{page: page}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
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
	if !strings.Contains(body, "Frete temporariamente indisponível para este carrinho.") ||
		!strings.Contains(body, "Frete indisponível") {
		t.Fatal("expected missing profile state")
	}
}

func TestCheckoutShippingGetShowsNoBoxState(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.Unavailable = true
	page.Message = "Não conseguimos calcular automaticamente o frete para este carrinho."
	service := &fakeCheckoutShippingService{page: page}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Não conseguimos calcular automaticamente o frete para este carrinho.") {
		t.Fatal("expected no-box state")
	}
}

func TestCheckoutShippingGetWithValidQuoteReturnsOK(t *testing.T) {
	service := &fakeCheckoutShippingService{page: checkoutShippingPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	assertCheckoutStepper(t, body, 2)
	for _, expected := range []string{"Como você quer receber?", "Modalidade de entrega", "Receber em casa", "Retirar no local", "PAC", "R$ 18,90", "5 dias úteis", "Continuar para revisão", `/static/js/checkout.js`, `data-checkout-form="true"`, `data-cep-lookup-endpoint="/api/cep"`, `data-checkout-mask="cep"`, `data-cep-status="true"`} {
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

func TestCheckoutShippingGetOffersPickupWithoutShippingQuote(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.SelectedMethod = ""
	service := &fakeCheckoutShippingService{page: page}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if service.lastMethod != "" {
		t.Fatalf("expected no delivery method before choice, got %q", service.lastMethod)
	}
	body := rec.Body.String()
	assertCheckoutStepper(t, body, 2)
	if !strings.Contains(body, "Retirar no local") || !strings.Contains(body, "Grátis") {
		t.Fatal("expected pickup option")
	}
}

func TestCheckoutShippingGetPrefillsProfileAddressWithoutQuoting(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.AddressForm = customers.CheckoutForm{Values: customers.CheckoutInput{CountryCode: customers.CountryCodeBR}}
	page.Quotes = nil
	service := &fakeCheckoutShippingService{page: page}
	profiles := &fakeAccountProfileService{profile: customerprofile.Profile{
		AuthUserID:  accountTestAuthProfile().ID,
		FullName:    "Cliente Auth",
		PostalCode:  "29100000",
		Street:      "Rua Perfil",
		Number:      "25",
		District:    "Centro",
		City:        "Vila Velha",
		State:       "ES",
		CountryCode: customers.CountryCodeBR,
	}, found: true}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/frete?delivery_method=shipping", nil)
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	checkoutShippingPageHandler(service, cookies, profiles).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Rua Perfil") || strings.Contains(body, "PAC") {
		t.Fatalf("expected profile address prefill without quote options, body=%s", body)
	}
	if profiles.getID != accountTestAuthProfile().ID {
		t.Fatalf("expected profile lookup by auth user id, got %q", profiles.getID)
	}
}

func TestCheckoutShippingPostAddressSyncsAuthenticatedProfile(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.AddressForm.Values = customers.CheckoutInput{
		PostalCode:  "29100000",
		Street:      "Rua Nova",
		Number:      "90",
		District:    "Centro",
		City:        "Vila Velha",
		State:       "ES",
		CountryCode: customers.CountryCodeBR,
	}
	service := &fakeCheckoutShippingService{page: page}
	profiles := &fakeAccountProfileService{profile: customerprofile.Profile{AuthUserID: accountTestAuthProfile().ID, FullName: "Joao Silva", Phone: "+5527999999999", CPF: "52998224725", CountryCode: customers.CountryCodeBR}, found: true}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutShippingAddressFormRequest(url.Values{
		"postal_code":  {"29100-000"},
		"street":       {"Rua Nova"},
		"number":       {"90"},
		"district":     {"Centro"},
		"city":         {"Vila Velha"},
		"state":        {"ES"},
		"country_code": {customers.CountryCodeBR},
	})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	saveShippingAddressHandler(service, cookies, "https://printlab.test", profiles).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/checkout/frete?delivery_method=shipping" {
		t.Fatalf("expected address redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !profiles.saveCalled || profiles.saved.Street != "Rua Nova" || profiles.saved.FullName != "Joao Silva" {
		t.Fatalf("expected authenticated shipping address to sync profile, got %#v", profiles.saved)
	}
}

func TestCheckoutShippingPostPickupRedirectsToReview(t *testing.T) {
	service := &fakeCheckoutShippingService{selectResult: shipping.SelectResult{Page: shipping.CheckoutShippingPage{Selected: true, SelectedMethod: shipping.DeliveryMethodPickup}}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutShippingFormRequest(url.Values{"delivery_method": {shipping.DeliveryMethodPickup}, "price": {"999999"}})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithShipping(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/checkout/revisao" {
		t.Fatalf("expected pickup redirect to review, got status %d location %q", rec.Code, rec.Header().Get("Location"))
	}
	if service.lastMethod != shipping.DeliveryMethodPickup || service.lastServiceCode != "" {
		t.Fatalf("expected pickup selection without service code, got method %q service %q", service.lastMethod, service.lastServiceCode)
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
	if rec.Header().Get("Location") != "/checkout/revisao" {
		t.Fatalf("expected selected redirect, got %q", rec.Header().Get("Location"))
	}
	if service.lastServiceCode != "1" {
		t.Fatalf("expected only service_code to be submitted to service, got %q", service.lastServiceCode)
	}
}

func TestCheckoutShippingPostInvalidServiceRerenders(t *testing.T) {
	page := checkoutShippingPageFixture()
	page.Message = "A opção selecionada não está mais disponível. Escolha uma cotação atual."
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
	if !strings.Contains(rec.Body.String(), "A opção selecionada não está mais disponível.") {
		t.Fatal("expected invalid selection message")
	}
}

func checkoutShippingFormRequest(values url.Values) *http.Request {
	body := strings.NewReader(values.Encode())
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/frete", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func checkoutShippingAddressFormRequest(values url.Values) *http.Request {
	body := strings.NewReader(values.Encode())
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/frete/endereco", body)
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
		SelectedMethod:      shipping.DeliveryMethodShipping,
		AddressForm: customers.CheckoutForm{
			Found: true,
			Values: customers.CheckoutInput{
				PostalCode:  "29100000",
				Street:      "Rua Um",
				Number:      "12",
				District:    "Centro",
				City:        "Vila Velha",
				State:       "ES",
				CountryCode: customers.CountryCodeBR,
			},
		},
		Quotes: []shipping.ShippingQuote{
			{
				Provider:     shipping.ProviderSuperFrete,
				ServiceCode:  "1",
				ServiceName:  "PAC",
				CarrierName:  "Correios",
				PriceCents:   1890,
				PriceBRL:     "R$ 18,90",
				DeliveryTime: "5 dias úteis",
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

	lastSelected     bool
	lastServiceCode  string
	lastMethod       string
	lastAddressInput customers.CheckoutInput
	saveAddressCalls int
}

func (s *fakeCheckoutShippingService) DeliveryPage(_ context.Context, _ []byte, method string) (shipping.CheckoutShippingPage, error) {
	s.pageCalls++
	s.lastSelected = false
	s.lastMethod = method
	if s.pageErr != nil {
		return shipping.CheckoutShippingPage{}, s.pageErr
	}

	return s.page, nil
}

func (s *fakeCheckoutShippingService) SelectDelivery(_ context.Context, _ []byte, method, serviceCode string) (shipping.SelectResult, error) {
	s.selectCalls++
	s.lastServiceCode = serviceCode
	s.lastMethod = method
	if s.selectErr != nil {
		return s.selectResult, s.selectErr
	}

	return s.selectResult, nil
}

func (s *fakeCheckoutShippingService) SaveAddress(_ context.Context, _ []byte, input customers.CheckoutInput) (shipping.CheckoutShippingPage, error) {
	s.saveAddressCalls++
	s.lastAddressInput = input
	if s.pageErr != nil {
		return s.page, s.pageErr
	}
	return s.page, nil
}

var _ checkoutShippingService = (*fakeCheckoutShippingService)(nil)
