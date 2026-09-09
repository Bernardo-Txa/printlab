package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
)

const cartItemID = "11111111-1111-1111-1111-111111111111"

func TestCartPageWithoutCookieReturnsEmptyState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/carrinho", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, nil, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Seu carrinho ainda esta vazio.") {
		t.Fatal("expected empty cart state")
	}
}

func TestCartAddValidItemRedirectsAndSetsCookie(t *testing.T) {
	service := &fakeCartService{cart: cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{Random: bytes.NewReader(make([]byte, cartdomain.TokenByteLength))})
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"produto-real"},
		"variant_slug": {"padrao"},
		"quantity":     {"2"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/carrinho" {
		t.Fatalf("expected redirect to /carrinho, got %q", got)
	}
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].Name != cartdomain.CookieName {
		t.Fatalf("expected cart cookie to be set, got %#v", rec.Result().Cookies())
	}
	if service.addCalls != 1 {
		t.Fatalf("expected add call, got %d", service.addCalls)
	}
	if service.lastAddInput.ProductSlug != "produto-real" || service.lastAddInput.VariantSlug != "padrao" || service.lastAddInput.Quantity != 2 {
		t.Fatalf("expected add input from form, got %#v", service.lastAddInput)
	}
}

func TestCartAddInvalidQuantityReturnsBadRequest(t *testing.T) {
	service := &fakeCartService{}
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"produto-real"},
		"quantity":     {"0"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if service.addCalls != 0 {
		t.Fatal("expected invalid quantity not to call service")
	}
}

func TestCartAddMissingProductDoesNotSetCookie(t *testing.T) {
	service := &fakeCartService{addErr: cartdomain.ErrInvalidProduct}
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"nao-existe"},
		"quantity":     {"1"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookie on failed add, got %#v", rec.Result().Cookies())
	}
}

func TestCartAddInvalidVariantDoesNotSetCookie(t *testing.T) {
	service := &fakeCartService{addErr: cartdomain.ErrInvalidVariant}
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"produto-real"},
		"variant_slug": {"invalida"},
		"quantity":     {"1"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookie on failed add, got %#v", rec.Result().Cookies())
	}
}

func TestCartUpdateQuantityRedirects(t *testing.T) {
	service := &fakeCartService{cartPointer: &cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := cartFormRequest("/carrinho/itens/"+cartItemID+"/quantidade", url.Values{"quantity": {"3"}})
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if service.updateCalls != 1 || service.lastItemID != cartItemID || service.lastQuantity != 3 {
		t.Fatalf("expected scoped update call, got service %#v", service)
	}
}

func TestCartRemoveRedirects(t *testing.T) {
	service := &fakeCartService{cartPointer: &cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}}
	cookies := cartdomain.NewCookieManager(cartdomain.CookieOptions{})
	req := cartFormRequest("/carrinho/itens/"+cartItemID+"/remover", nil)
	req.Header.Set("Origin", "https://printlab.test")
	addValidCartCookie(t, cookies, req)
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, cookies, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if service.removeCalls != 1 || service.lastItemID != cartItemID {
		t.Fatalf("expected scoped remove call, got service %#v", service)
	}
}

func TestCartMutationRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeCartService{}
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"produto-real"},
		"quantity":     {"1"},
	})
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if service.addCalls != 0 {
		t.Fatal("expected cross-site mutation not to call service")
	}
}

func TestValidMutationSourceAllowsConfiguredSiteURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://preview.printlab.test/carrinho/adicionar", nil)
	req.Header.Set("Origin", "https://printlab.test")

	if !validMutationSource(req, "https://printlab.test") {
		t.Fatal("expected configured SITE_URL origin to be allowed")
	}
}

func TestSecureCartCookiesForProductionRuntime(t *testing.T) {
	if !secureCartCookies(config.Config{AppEnv: "production"}) {
		t.Fatal("expected production APP_ENV to use secure cookies")
	}
	if !secureCartCookies(config.Config{VercelEnv: "production"}) {
		t.Fatal("expected production Vercel runtime to use secure cookies")
	}
	if !secureCartCookies(config.Config{SiteURL: "https://printlab.test"}) {
		t.Fatal("expected https SITE_URL to use secure cookies")
	}
	if secureCartCookies(config.Config{SiteURL: "http://localhost:8080"}) {
		t.Fatal("expected local http SITE_URL not to force secure cookies")
	}
}

func cartFormRequest(target string, values url.Values) *http.Request {
	body := strings.NewReader("")
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test"+target, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func addValidCartCookie(t *testing.T, cookies *cartdomain.CookieManager, req *http.Request) {
	t.Helper()

	token, err := cookies.GenerateToken()
	if err != nil {
		t.Fatalf("expected token to be generated, got %v", err)
	}
	req.AddCookie(cookies.Cookie(token, time.Now().Add(cartdomain.TTL)))
}

func newTestHandlerWithCart(t *testing.T, service cartService, cookies *cartdomain.CookieManager, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServices(db, nil, service, nil, nil, cookies, siteURL)
}

type fakeCartService struct {
	view        cartdomain.CartView
	cart        cartdomain.Cart
	cartPointer *cartdomain.Cart

	viewErr   error
	addErr    error
	updateErr error
	removeErr error

	addCalls    int
	updateCalls int
	removeCalls int

	lastAddInput cartdomain.AddItemInput
	lastItemID   string
	lastQuantity int
}

func (s *fakeCartService) View(_ context.Context, _ []byte) (cartdomain.CartView, error) {
	if s.viewErr != nil {
		return cartdomain.CartView{}, s.viewErr
	}

	return s.view, nil
}

func (s *fakeCartService) CheckoutCart(_ context.Context, _ []byte) (cartdomain.Cart, cartdomain.CartView, error) {
	if s.viewErr != nil {
		return cartdomain.Cart{}, cartdomain.CartView{}, s.viewErr
	}

	return s.cart, s.view, nil
}

func (s *fakeCartService) Renew(_ context.Context, _ string) (cartdomain.Cart, error) {
	if s.cart.ID == "" {
		s.cart = cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}
	}

	return s.cart, nil
}

func (s *fakeCartService) Add(_ context.Context, _ []byte, input cartdomain.AddItemInput) (cartdomain.Cart, error) {
	s.addCalls++
	s.lastAddInput = input
	if s.addErr != nil {
		return cartdomain.Cart{}, s.addErr
	}

	if s.cart.ID == "" {
		s.cart = cartdomain.Cart{ID: "cart-1", ExpiresAt: time.Now().Add(cartdomain.TTL)}
	}

	return s.cart, nil
}

func (s *fakeCartService) UpdateQuantity(_ context.Context, _ []byte, itemID string, quantity int) (*cartdomain.Cart, error) {
	s.updateCalls++
	s.lastItemID = itemID
	s.lastQuantity = quantity
	if s.updateErr != nil {
		return nil, s.updateErr
	}

	return s.cartPointer, nil
}

func (s *fakeCartService) RemoveItem(_ context.Context, _ []byte, itemID string) (*cartdomain.Cart, error) {
	s.removeCalls++
	s.lastItemID = itemID
	if s.removeErr != nil {
		return nil, s.removeErr
	}

	return s.cartPointer, nil
}

var _ cartService = (*fakeCartService)(nil)

func TestCartHandlerErrorsDoNotLeakInternalDetails(t *testing.T) {
	service := &fakeCartService{addErr: errors.New("postgres password=secret")}
	req := cartFormRequest("/carrinho/adicionar", url.Values{
		"product_slug": {"produto-real"},
		"quantity":     {"1"},
	})
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	newTestHandlerWithCart(t, service, nil, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	body := strings.ToLower(rec.Body.String())
	for _, term := range []string{"postgres", "password", "secret"} {
		if strings.Contains(body, term) {
			t.Fatalf("expected response not to leak %q", term)
		}
	}
}
