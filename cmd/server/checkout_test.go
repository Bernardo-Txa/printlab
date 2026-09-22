package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/database"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
)

func TestCheckoutDetailsGetWithoutCartRedirectsToCart(t *testing.T) {
	service := &fakeCheckoutDetailsService{}
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/carrinho" {
		t.Fatalf("expected redirect to /carrinho, got %q", rec.Header().Get("Location"))
	}
	if service.pageCalls != 0 {
		t.Fatal("expected missing cart not to call checkout service")
	}
}

func TestCheckoutDetailsGetWithCartReturnsOK(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Etapa 1 - Seus dados") || !strings.Contains(body, "Resumo do carrinho") {
		t.Fatal("expected checkout details form and cart summary")
	}
	if !strings.Contains(body, "Usamos estes dados somente para preparar sua compra") {
		t.Fatal("expected privacy copy")
	}
	for _, expected := range []string{
		`data-checkout-form="true"`,
		`data-cep-lookup-endpoint="/api/cep"`,
		`data-checkout-mask="cpf"`,
		`data-checkout-mask="phone"`,
		`data-checkout-mask="cep"`,
		`data-cep-status="true"`,
		`/static/js/checkout.js`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected checkout UX enhancement marker %q", expected)
		}
	}
	for _, forbiddenField := range []string{`name="unit_price"`, `name="subtotal"`, `name="total"`, `name="product_name"`} {
		if strings.Contains(body, forbiddenField) {
			t.Fatalf("expected checkout form not to include authoritative field %s", forbiddenField)
		}
	}
}

func TestCheckoutDetailsPostValidRedirects(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if rec.Header().Get("Location") != "/checkout/frete" {
		t.Fatalf("expected saved redirect, got %q", rec.Header().Get("Location"))
	}
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].Name != cartdomain.CookieName {
		t.Fatalf("expected renewed cart cookie, got %#v", rec.Result().Cookies())
	}
	if service.saveCalls != 1 {
		t.Fatalf("expected one save call, got %d", service.saveCalls)
	}
	if service.lastInput.CPF != "529.982.247-25" || service.lastInput.CountryCode != customers.CountryCodeBR {
		t.Fatalf("expected submitted customer fields, got %#v", service.lastInput)
	}
}

func TestCheckoutDetailsGetKeepsExistingCartDataBeforeProfile(t *testing.T) {
	page := checkoutPageFixture()
	page.Form.Values = validCheckoutInputForHandler(func(input *customers.CheckoutInput) {
		input.FullName = "Carrinho Atual"
	})
	page.Form.Found = true
	service := &fakeCheckoutDetailsService{page: page}
	profiles := &fakeAccountProfileService{profile: customerprofile.Profile{FullName: "Perfil Cliente"}, found: true}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	checkoutDetailsPageHandler(service, cookies, profiles, &fakeOrderReviewService{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Carrinho Atual") || strings.Contains(body, "Perfil Cliente") {
		t.Fatal("expected existing cart data to have priority over profile")
	}
	if profiles.getID != "" {
		t.Fatalf("expected profile not to be loaded when cart has details, got %q", profiles.getID)
	}
}

func TestCheckoutDetailsGetUsesCustomerProfileFallback(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	profiles := &fakeAccountProfileService{profile: accountSavedProfile(), found: true}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	checkoutDetailsPageHandler(service, cookies, profiles, &fakeOrderReviewService{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Perfil Salvo") || !strings.Contains(body, `value="cliente@example.com"`) {
		t.Fatal("expected checkout fallback from customer profile and auth email")
	}
}

func TestCheckoutDetailsGetUsesLatestOrderFallback(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	ordersService := &fakeOrderReviewService{snapshotFound: true, snapshot: ordersdomain.CustomerSnapshot{
		FullName: "Pedido Checkout", Phone: "+5527999999999", CPF: "52998224725", PostalCode: "29100000", Street: "Rua Pedido", Number: "8", District: "Centro", City: "Vila Velha", State: "ES", CountryCode: customers.CountryCodeBR,
	}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	checkoutDetailsPageHandler(service, cookies, &fakeAccountProfileService{}, ordersService).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ordersService.lastSnapshotCustomerID != accountTestAuthProfile().ID {
		t.Fatalf("expected latest order lookup by auth id, got %q", ordersService.lastSnapshotCustomerID)
	}
	if !strings.Contains(rec.Body.String(), "Pedido Checkout") {
		t.Fatal("expected checkout fallback from latest order")
	}
}

func TestCheckoutDetailsGetUsesAuthNameEmailFallback(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	checkoutDetailsPageHandler(service, cookies, &fakeAccountProfileService{}, &fakeOrderReviewService{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{`value="Cliente Auth"`, `value="cliente@example.com"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected auth fallback to contain %q", expected)
		}
	}
}

func TestCheckoutDetailsGuestRemainsUnchanged(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: checkoutPageFixture()}
	profiles := &fakeAccountProfileService{profile: accountSavedProfile(), found: true}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/checkout/dados", nil)
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	checkoutDetailsPageHandler(service, cookies, profiles, &fakeOrderReviewService{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if profiles.getID != "" || strings.Contains(rec.Body.String(), "Perfil Salvo") {
		t.Fatal("expected guest checkout not to use customer profile")
	}
}

func TestCheckoutDetailsPostAuthenticatedUpdatesProfile(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: customers.CheckoutPage{Cart: checkoutPageFixture().Cart, Form: customers.CheckoutForm{Values: validCheckoutInputForHandler(nil)}}}
	profiles := &fakeAccountProfileService{}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()))
	rec := httptest.NewRecorder()

	saveCheckoutDetailsHandler(service, cookies, "https://printlab.test", profiles).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if !profiles.saveCalled || profiles.saved.AuthUserID != accountTestAuthProfile().ID || profiles.saved.FullName != "Joao Silva" {
		t.Fatalf("expected authenticated checkout to sync profile, got %#v", profiles.saved)
	}
}

func TestCheckoutDetailsPostGuestDoesNotUpdateProfile(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: customers.CheckoutPage{Cart: checkoutPageFixture().Cart, Form: customers.CheckoutForm{Values: validCheckoutInputForHandler(nil)}}}
	profiles := &fakeAccountProfileService{}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	saveCheckoutDetailsHandler(service, cookies, "https://printlab.test", profiles).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if profiles.saveCalled {
		t.Fatal("expected guest checkout not to sync profile")
	}
}

func TestCheckoutDetailsPostProfileSyncErrorLogsSafeEventAndRedirects(t *testing.T) {
	service := &fakeCheckoutDetailsService{page: customers.CheckoutPage{Cart: checkoutPageFixture().Cart, Form: customers.CheckoutForm{Values: validCheckoutInputForHandler(nil)}}}
	profiles := &fakeAccountProfileService{saveErr: errors.New("db failed for Joao Silva cliente@example.com 52998224725 Rua Um")}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	req = req.WithContext(requestIDContext(customerauth.WithProfile(req.Context(), accountTestAuthProfile()), "0123456789abcdef"))
	rec := httptest.NewRecorder()
	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})

	saveCheckoutDetailsHandler(service, cookies, "https://printlab.test", profiles).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/checkout/frete" {
		t.Fatalf("expected checkout to continue, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	logText := logs.String()
	if !strings.Contains(logText, "event=customer_profile_sync_failed") || !strings.Contains(logText, "request_id=0123456789abcdef") {
		t.Fatalf("expected safe sync failure event, got %q", logText)
	}
	for _, leaked := range []string{"Joao", "cliente@example.com", "52998224725", "Rua Um"} {
		if strings.Contains(logText, leaked) {
			t.Fatalf("expected profile sync log not to leak %q", leaked)
		}
	}
}

func TestCheckoutDetailsPostInvalidCPFRerendersForm(t *testing.T) {
	service := &fakeCheckoutDetailsService{
		saveErr: customers.ErrInvalidDetails,
		savePage: customers.CheckoutPage{
			Cart: checkoutPageFixture().Cart,
			Form: customers.CheckoutForm{
				Values: validCheckoutInputForHandler(func(input *customers.CheckoutInput) {
					input.CPF = "529.982.247-24"
				}),
				Errors: customers.FieldErrors{"cpf": "Informe um CPF valido."},
			},
		},
	}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	form := validCheckoutForm()
	form.Set("cpf", "529.982.247-24")
	req := checkoutFormRequest(form)
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Informe um CPF valido.") || !strings.Contains(body, `value="Joao Silva"`) {
		t.Fatal("expected validation error and safe submitted values")
	}
}

func TestCheckoutDetailsPostInvalidAddressRerendersForm(t *testing.T) {
	service := &fakeCheckoutDetailsService{
		saveErr: customers.ErrInvalidDetails,
		savePage: customers.CheckoutPage{
			Cart: checkoutPageFixture().Cart,
			Form: customers.CheckoutForm{
				Values: validCheckoutInputForHandler(func(input *customers.CheckoutInput) {
					input.PostalCode = "2910A000"
				}),
				Errors: customers.FieldErrors{"postal_code": "Informe um CEP valido."},
			},
		},
	}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	form := validCheckoutForm()
	form.Set("postal_code", "2910A000")
	req := checkoutFormRequest(form)
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Informe um CEP valido.") {
		t.Fatal("expected address validation error")
	}
}

func TestCheckoutDetailsPostRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeCheckoutDetailsService{}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://evil.example")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if service.saveCalls != 0 {
		t.Fatal("expected cross-site request not to call checkout service")
	}
}

func TestCheckoutDetailsErrorsDoNotLeakInternalDetails(t *testing.T) {
	service := &fakeCheckoutDetailsService{saveErr: errors.New("postgres DATABASE_URL password=secret")}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := checkoutFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCheckout(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	body := strings.ToLower(rec.Body.String())
	for _, term := range []string{"postgres", "database_url", "password", "secret", "52998224725"} {
		if strings.Contains(body, term) {
			t.Fatalf("expected response not to leak %q", term)
		}
	}
}

func checkoutFormRequest(values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/checkout/dados", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func validCheckoutForm() url.Values {
	return url.Values{
		"full_name":    {"Joao Silva"},
		"email":        {"joao@example.com"},
		"phone":        {"(27) 99999-9999"},
		"cpf":          {"529.982.247-25"},
		"postal_code":  {"29100-000"},
		"street":       {"Rua Um"},
		"number":       {"12A"},
		"complement":   {"Apto 302"},
		"district":     {"Centro"},
		"city":         {"Vila Velha"},
		"state":        {"ES"},
		"country_code": {customers.CountryCodeBR},
	}
}

func validCheckoutInputForHandler(modify func(*customers.CheckoutInput)) customers.CheckoutInput {
	input := customers.CheckoutInput{
		FullName:    "Joao Silva",
		Email:       "joao@example.com",
		Phone:       "(27) 99999-9999",
		CPF:         "529.982.247-25",
		PostalCode:  "29100-000",
		Street:      "Rua Um",
		Number:      "12A",
		Complement:  "Apto 302",
		District:    "Centro",
		City:        "Vila Velha",
		State:       "ES",
		CountryCode: customers.CountryCodeBR,
	}
	if modify != nil {
		modify(&input)
	}

	return input
}

func checkoutPageFixture() customers.CheckoutPage {
	return customers.CheckoutPage{
		Cart: cartdomain.CartView{
			Lines: []cartdomain.CartLine{
				{
					ID:          "item-1",
					ProductName: "Produto Real",
					Quantity:    2,
					SubtotalBRL: "R$ 79,80",
					Available:   true,
				},
			},
			SubtotalBRL: "R$ 79,80",
		},
		Form: customers.CheckoutForm{
			Values: customers.CheckoutInput{CountryCode: customers.CountryCodeBR},
		},
	}
}

func newTestHandlerWithCheckout(t *testing.T, service checkoutDetailsService, cookies *cartdomain.CookieManager, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServices(db, nil, nil, service, nil, cookies, nil, siteURL)
}

type fakeCheckoutDetailsService struct {
	page     customers.CheckoutPage
	savePage customers.CheckoutPage
	pageErr  error
	saveErr  error

	pageCalls int
	saveCalls int
	lastInput customers.CheckoutInput
}

func (s *fakeCheckoutDetailsService) Page(_ context.Context, _ []byte, saved bool) (customers.CheckoutPage, error) {
	s.pageCalls++
	if s.pageErr != nil {
		return customers.CheckoutPage{}, s.pageErr
	}
	page := s.page
	page.Form.Saved = saved
	return page, nil
}

func (s *fakeCheckoutDetailsService) Save(_ context.Context, _ []byte, input customers.CheckoutInput) (customers.SaveResult, error) {
	s.saveCalls++
	s.lastInput = input
	if s.saveErr != nil {
		return customers.SaveResult{Page: s.savePage}, s.saveErr
	}
	return customers.SaveResult{Page: s.page, ExpiresAt: time.Now().Add(cartdomain.TTL)}, nil
}

var _ checkoutDetailsService = (*fakeCheckoutDetailsService)(nil)
