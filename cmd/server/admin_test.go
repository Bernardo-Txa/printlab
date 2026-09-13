package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/database"
)

func TestAdminLoginPage(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, &fakeAdminPanelService{available: true}, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	body := rec.Body.String()
	if !strings.Contains(body, "Acesso administrativo") || !strings.Contains(body, `autocomplete="current-password"`) {
		t.Fatal("expected admin login form")
	}
}

func TestAdminLoginUnavailableWhenConfigMissing(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, nil, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	if strings.Contains(rec.Body.String(), "SUPABASE") || strings.Contains(rec.Body.String(), "ADMIN_SUPABASE_USER_ID") {
		t.Fatal("expected unavailable page not to reveal missing environment variables")
	}
}

func TestAdminLoginValidRedirectsAndSetsCookie(t *testing.T) {
	expiresAt := time.Now().Add(admindomain.SessionTTL)
	service := &fakeAdminPanelService{
		available: true,
		loginResult: admindomain.LoginResult{
			Token:     mustAdminToken(t),
			ExpiresAt: expiresAt,
		},
	}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	form := url.Values{"email": {"admin@example.com"}, "password": {"correct-password"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin" {
		t.Fatalf("expected redirect to /admin, got %q", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != admindomain.CookieName {
		t.Fatalf("expected admin session cookie, got %#v", cookies)
	}
	if cookies[0].Path != "/admin" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected safe admin cookie attributes, got %#v", cookies[0])
	}
	if service.loginEmail != "admin@example.com" || service.loginPassword != "correct-password" {
		t.Fatalf("expected login credentials to be forwarded only to service, got %q/%q", service.loginEmail, service.loginPassword)
	}
}

func TestAdminLoginInvalidCredentialsShowsGenericMessageAndNoCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true, loginErr: admindomain.ErrInvalidCredentials}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	form := url.Values{"email": {"other@example.com"}, "password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), admindomain.InvalidCredentialsMessage) {
		t.Fatal("expected generic invalid credentials message")
	}
	if strings.Contains(rec.Body.String(), "não autorizado") || strings.Contains(rec.Body.String(), "wrong-password") || len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected no authorization details, password, or cookie")
	}
}

func TestAdminLoginRejectsOpaqueOriginWithSameOriginReferer(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader("email=admin@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "null")
	req.Header.Set("Referer", "https://printlab.test/admin/login")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.loginCalled {
		t.Fatal("expected opaque login with referer not to call service")
	}
}

func TestAdminLoginRejectsOpaqueOriginWithoutReferer(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader("email=admin@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "null")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.loginCalled {
		t.Fatal("expected opaque login without referer not to call service")
	}
}

func TestAdminLoginRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader("email=admin@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.loginCalled {
		t.Fatal("expected cross-site login not to call service")
	}
}

func TestAdminDashboardRedirectsWithoutSession(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrUnauthenticated}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
	if !service.resolveCalled {
		t.Fatal("expected session guard")
	}
}

func TestAdminDashboardRedirectsToLoginWhenAdminUnavailable(t *testing.T) {
	handler := newTestHandlerWithAdmin(t, nil, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
}

func TestAdminDashboardRendersForValidSession(t *testing.T) {
	service := &fakeAdminPanelService{
		available: true,
		dashboard: admindomain.Dashboard{
			PendingPayment:        1,
			PaidWaitingProduction: 2,
			InProduction:          3,
			WaitingShipment:       4,
		},
	}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	assertAdminHeaders(t, rec)
	body := rec.Body.String()
	for _, expected := range []string{"Aguardando pagamento", ">1<", "Pagos aguardando produção", ">2<", "Em produção", ">3<", "Aguardando envio", ">4<"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected dashboard body to contain %q", expected)
		}
	}
	for _, forbidden := range []string{"CPF", "transaction_nsu", "checkout_url", "admin@example.com", "Rua "} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected dashboard not to expose %q", forbidden)
		}
	}
}

func TestAdminOrdersRendersListWithoutPII(t *testing.T) {
	createdAt := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	page := admindomain.PrepareOrderListPage(admindomain.OrderListFilter{Status: admindomain.OrderListStatusWaitingShipment, Page: 1}, []admindomain.OrderListItem{
		{
			ID:               "11111111-1111-1111-1111-111111111111",
			OrderNumber:      1001,
			OrderStatus:      admindomain.OrderStatusPaid,
			ProductionStatus: admindomain.ProductionStatusCompleted,
			ShippingStatus:   admindomain.ShippingStatusWaiting,
			CreatedAt:        createdAt,
			TotalCents:       12500,
			ItemCount:        1,
			UnitCount:        2,
		},
	})
	service := &fakeAdminPanelService{available: true, orders: page}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/pedidos?status=waiting_shipment", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !service.listOrdersCalled || service.listFilter.Status != admindomain.OrderListStatusWaitingShipment {
		t.Fatalf("expected list filter to be forwarded, got %#v", service.listFilter)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Pedidos", "#1001", "Aguardando envio", "R$"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected orders page to contain %q", expected)
		}
	}
	for _, forbidden := range []string{"CPF", "admin@example.com", "transaction_nsu", "checkout_url", "invoice_slug", "Rua "} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected orders list not to expose %q", forbidden)
		}
	}
}

func TestAdminOrderDetailRendersPrivateDataAndActions(t *testing.T) {
	detail := admindomain.OrderDetail{
		ID:                 "11111111-1111-1111-1111-111111111111",
		PublicTrackingID:   "22222222-2222-2222-2222-222222222222",
		OrderNumber:        1001,
		OrderStatus:        admindomain.OrderStatusPaid,
		ProductionStatus:   admindomain.ProductionStatusWaiting,
		ShippingStatus:     admindomain.ShippingStatusWaiting,
		CreatedAt:          time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
		ShippingPriceCents: 2500,
		TotalCents:         12500,
		Customer: admindomain.OrderCustomer{
			FullName: "Cliente Teste",
			Email:    "cliente@example.com",
			Phone:    "+5511999999999",
			CPF:      "12345678909",
		},
		Address: admindomain.OrderAddress{
			LineOne:     "Rua Teste, 123",
			LineTwo:     "Centro - Sao Paulo/SP - CEP 01001-000",
			CountryCode: "BR",
		},
		Shipping: admindomain.OrderShipping{
			ServiceName:        "PAC",
			DeliveryTime:       "5 dias uteis",
			ShippingBoxName:    "Caixa P",
			PackageWeightLabel: "500 g",
			DimensionsLabel:    "200 x 100 x 80 mm",
			PriceBRL:           "R$ 25,00",
		},
		Payment: admindomain.OrderPayment{Available: true, Status: "paid", StatusLabel: "Pago"},
		Items: []admindomain.OrderDetailItem{
			{ProductName: "Produto", Quantity: 1, UnitPriceBRL: "R$ 100,00", LineTotalBRL: "R$ 100,00"},
		},
	}
	admindomain.PrepareOrderDetail(&detail)
	service := &fakeAdminPanelService{available: true, order: detail}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/pedidos/"+detail.ID, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Cliente Teste", "cliente@example.com", "12345678909", "Iniciar producao", "/acompanhar/22222222-2222-2222-2222-222222222222"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected detail to contain %q", expected)
		}
	}
	for _, forbidden := range []string{"transaction_nsu", "checkout_url", "invoice_slug"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected detail not to expose payment identifier %q", forbidden)
		}
	}
}

func TestAdminProductionStatusValidatesOriginAndUsesSessionActor(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	orderID := "11111111-1111-1111-1111-111111111111"
	form := url.Values{"production_status": {admindomain.ProductionStatusInProduction}}
	req := httptest.NewRequest(http.MethodPost, "/admin/pedidos/"+orderID+"/producao", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/pedidos/"+orderID+"?ok=producao" {
		t.Fatalf("expected success redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.productionCalled || service.productionOrderID != orderID || service.productionStatus != admindomain.ProductionStatusInProduction {
		t.Fatalf("expected production mutation call, got %#v", service)
	}
	if service.productionActorID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected session actor id, got %q", service.productionActorID)
	}
}

func TestAdminProductionStatusRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	orderID := "11111111-1111-1111-1111-111111111111"
	req := httptest.NewRequest(http.MethodPost, "/admin/pedidos/"+orderID+"/producao", strings.NewReader("production_status=in_production"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.productionCalled {
		t.Fatal("expected cross-site production mutation not to call service")
	}
}

func TestAdminShippingStatusConflictRedirectsToDetail(t *testing.T) {
	service := &fakeAdminPanelService{available: true, shippingErr: admindomain.ErrTransitionConflict}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	orderID := "11111111-1111-1111-1111-111111111111"
	req := httptest.NewRequest(http.MethodPost, "/admin/pedidos/"+orderID+"/envio", strings.NewReader("shipping_status=shipped"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/pedidos/"+orderID+"?erro=transicao" {
		t.Fatalf("expected conflict redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAdminProductsRequireSession(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrUnauthenticated}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/produtos", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAdminCreateProductSameOriginCallsService(t *testing.T) {
	service := &fakeAdminPanelService{available: true, createProductID: "22222222-2222-2222-2222-222222222222"}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	form := url.Values{
		"name":      {"Produto Admin"},
		"slug":      {"produto-admin"},
		"price":     {"39,90"},
		"is_active": {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/admin/produtos", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://printlab.test")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/produtos/22222222-2222-2222-2222-222222222222?ok=salvo" {
		t.Fatalf("expected create redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.createProductCalled || service.createProductForm.PriceBRL != "39,90" {
		t.Fatalf("expected create product call, got %#v", service.createProductForm)
	}
}

func TestAdminCreateProductRejectsMaliciousOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/admin/produtos", strings.NewReader("name=Produto"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.createProductCalled {
		t.Fatal("service must not be called for cross-site origin")
	}
}

func TestAdminCreateProductRejectsOpaqueOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/admin/produtos", strings.NewReader("name=Produto"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "null")
	req.Header.Set("Referer", "https://printlab.test/admin/produtos/novo")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.createProductCalled {
		t.Fatal("service must not be called for opaque origin")
	}
}

func TestAdminRecipeMutationDoesNotUseGET(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/produtos/11111111-1111-1111-1111-111111111111/variantes/22222222-2222-2222-2222-222222222222/receita", nil)
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found for GET mutation route, got %d", rec.Code)
	}
	if service.addRecipeCalled {
		t.Fatal("GET must not mutate recipe")
	}
}

func TestAdminDashboardExpiredSessionRedirectsAndClearsCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrSessionExpired}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected expired admin cookie to be cleared, got %#v", rec.Result().Cookies())
	}
}

func TestAdminDashboardRandomCookieRedirects(t *testing.T) {
	service := &fakeAdminPanelService{available: true, resolveErr: admindomain.ErrUnauthenticated}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: "random"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAdminRedirectToLogin(t, rec)
}

func TestAdminLogoutPostValidatesOriginDeletesSessionAndClearsCookie(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.Header.Set("Origin", "https://printlab.test")
	req.AddCookie(&http.Cookie{Name: admindomain.CookieName, Value: mustAdminToken(t)})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.logoutCalled {
		t.Fatal("expected logout to call service")
	}
	if len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected admin cookie to be cleared, got %#v", rec.Result().Cookies())
	}
}

func TestAdminLogoutRejectsCrossSiteOrigin(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if service.logoutCalled {
		t.Fatal("expected cross-site logout not to call service")
	}
}

func TestAdminLogoutGetDoesNotLogout(t *testing.T) {
	service := &fakeAdminPanelService{available: true}
	handler := newTestHandlerWithAdmin(t, service, "https://printlab.test")
	req := httptest.NewRequest(http.MethodGet, "/admin/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected method not allowed, got %d", rec.Code)
	}
	if service.logoutCalled {
		t.Fatal("expected GET logout not to call service")
	}
}

func newTestHandlerWithAdmin(t *testing.T, adminPanel adminPanelService, siteURL string) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServicesAndOrders(db, nil, nil, nil, nil, nil, nil, nil, nil, adminPanel, siteURL)
}

func assertAdminHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get("Cache-Control") != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", rec.Header().Get("Cache-Control"))
	}
	if robots := rec.Header().Get("X-Robots-Tag"); !strings.Contains(robots, "noindex") || !strings.Contains(robots, "nofollow") || !strings.Contains(robots, "noarchive") {
		t.Fatalf("expected noindex robots header, got %q", robots)
	}
	if rec.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatalf("expected same-origin referrer policy, got %q", rec.Header().Get("Referrer-Policy"))
	}
}

func assertAdminRedirectToLogin(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("expected redirect to login, got %q", rec.Header().Get("Location"))
	}
	assertAdminHeaders(t, rec)
}

func mustAdminToken(t *testing.T) string {
	t.Helper()

	token, err := admindomain.NewTokenManager(admindomain.CookieOptions{}).GenerateToken()
	if err != nil {
		t.Fatalf("expected admin token, got %v", err)
	}
	return token
}

type fakeAdminPanelService struct {
	available           bool
	loginCalled         bool
	loginEmail          string
	loginPassword       string
	loginResult         admindomain.LoginResult
	loginErr            error
	resolveCalled       bool
	resolveErr          error
	logoutCalled        bool
	logoutErr           error
	dashboard           admindomain.Dashboard
	dashboardErr        error
	listOrdersCalled    bool
	listFilter          admindomain.OrderListFilter
	orders              admindomain.OrderListPage
	ordersErr           error
	getOrderCalled      bool
	getOrderID          string
	order               admindomain.OrderDetail
	orderErr            error
	productionCalled    bool
	productionOrderID   string
	productionStatus    string
	productionActorID   string
	productionErr       error
	shippingCalled      bool
	shippingOrderID     string
	shippingStatus      string
	shippingActorID     string
	shippingErr         error
	createProductCalled bool
	createProductID     string
	createProductForm   admindomain.AdminProductForm
	createProductErr    error
	addRecipeCalled     bool
}

func (s *fakeAdminPanelService) Available() bool {
	return s != nil && s.available
}

func (s *fakeAdminPanelService) Login(_ context.Context, email string, password string) (admindomain.LoginResult, error) {
	s.loginCalled = true
	s.loginEmail = email
	s.loginPassword = password
	if s.loginErr != nil {
		return admindomain.LoginResult{}, s.loginErr
	}
	return s.loginResult, nil
}

func (s *fakeAdminPanelService) ResolveSession(context.Context, *http.Request) (admindomain.Session, error) {
	s.resolveCalled = true
	if s.resolveErr != nil {
		return admindomain.Session{}, s.resolveErr
	}
	return admindomain.Session{AuthUserID: "11111111-1111-1111-1111-111111111111"}, nil
}

func (s *fakeAdminPanelService) Logout(context.Context, *http.Request) error {
	s.logoutCalled = true
	if s.logoutErr != nil {
		return s.logoutErr
	}
	return nil
}

func (s *fakeAdminPanelService) Dashboard(context.Context) (admindomain.Dashboard, error) {
	if s.dashboardErr != nil {
		return admindomain.Dashboard{}, s.dashboardErr
	}
	return s.dashboard, nil
}

func (s *fakeAdminPanelService) ListOrders(_ context.Context, filter admindomain.OrderListFilter) (admindomain.OrderListPage, error) {
	s.listOrdersCalled = true
	s.listFilter = filter
	if s.ordersErr != nil {
		return admindomain.OrderListPage{}, s.ordersErr
	}
	return s.orders, nil
}

func (s *fakeAdminPanelService) GetOrder(_ context.Context, orderID string) (admindomain.OrderDetail, error) {
	s.getOrderCalled = true
	s.getOrderID = orderID
	if s.orderErr != nil {
		return admindomain.OrderDetail{}, s.orderErr
	}
	return s.order, nil
}

func (s *fakeAdminPanelService) ChangeProductionStatus(_ context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	s.productionCalled = true
	s.productionOrderID = orderID
	s.productionStatus = targetStatus
	s.productionActorID = actorAuthUserID
	return s.productionErr
}

func (s *fakeAdminPanelService) ChangeShippingStatus(_ context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	s.shippingCalled = true
	s.shippingOrderID = orderID
	s.shippingStatus = targetStatus
	s.shippingActorID = actorAuthUserID
	return s.shippingErr
}

func (s *fakeAdminPanelService) ListAdminProducts(context.Context, admindomain.AdminProductListFilter) (admindomain.AdminProductListPage, error) {
	return admindomain.AdminProductListPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminProduct(context.Context) (admindomain.AdminProductFormPage, error) {
	return admindomain.AdminProductFormPage{}, nil
}

func (s *fakeAdminPanelService) GetAdminProduct(context.Context, string) (admindomain.AdminProductFormPage, error) {
	return admindomain.AdminProductFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminProduct(_ context.Context, form admindomain.AdminProductForm) (string, admindomain.AdminProductFormPage, error) {
	s.createProductCalled = true
	s.createProductForm = form
	if s.createProductErr != nil {
		return "", admindomain.AdminProductFormPage{}, s.createProductErr
	}
	return s.createProductID, admindomain.AdminProductFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminProduct(context.Context, string, admindomain.AdminProductForm) (admindomain.AdminProductFormPage, error) {
	return admindomain.AdminProductFormPage{}, nil
}

func (s *fakeAdminPanelService) ListAdminCategories(context.Context) (admindomain.AdminCategoryListPage, error) {
	return admindomain.AdminCategoryListPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminCategory() admindomain.AdminCategoryFormPage {
	return admindomain.AdminCategoryFormPage{}
}

func (s *fakeAdminPanelService) GetAdminCategory(context.Context, string) (admindomain.AdminCategoryFormPage, error) {
	return admindomain.AdminCategoryFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminCategory(context.Context, admindomain.AdminCategoryForm) (string, admindomain.AdminCategoryFormPage, error) {
	return "", admindomain.AdminCategoryFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminCategory(context.Context, string, admindomain.AdminCategoryForm) (admindomain.AdminCategoryFormPage, error) {
	return admindomain.AdminCategoryFormPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminVariant(context.Context, string) (admindomain.AdminVariantFormPage, error) {
	return admindomain.AdminVariantFormPage{}, nil
}

func (s *fakeAdminPanelService) GetAdminVariant(context.Context, string, string) (admindomain.AdminVariantFormPage, error) {
	return admindomain.AdminVariantFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminVariant(context.Context, string, admindomain.AdminVariantForm) (string, admindomain.AdminVariantFormPage, error) {
	return "", admindomain.AdminVariantFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminVariant(context.Context, string, string, admindomain.AdminVariantForm) (admindomain.AdminVariantFormPage, error) {
	return admindomain.AdminVariantFormPage{}, nil
}

func (s *fakeAdminPanelService) AddAdminRecipeComponent(context.Context, string, string, admindomain.AdminRecipeForm) error {
	s.addRecipeCalled = true
	return nil
}

func (s *fakeAdminPanelService) UpdateAdminRecipeComponent(context.Context, string, string, string, admindomain.AdminRecipeForm) error {
	return nil
}

func (s *fakeAdminPanelService) RemoveAdminRecipeComponent(context.Context, string, string, string) error {
	return nil
}

func (s *fakeAdminPanelService) ListAdminMaterials(context.Context) (admindomain.AdminMaterialListPage, error) {
	return admindomain.AdminMaterialListPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminMaterial() admindomain.AdminMaterialFormPage {
	return admindomain.AdminMaterialFormPage{}
}

func (s *fakeAdminPanelService) GetAdminMaterial(context.Context, string) (admindomain.AdminMaterialFormPage, error) {
	return admindomain.AdminMaterialFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminMaterial(context.Context, admindomain.AdminMaterialForm) (string, admindomain.AdminMaterialFormPage, error) {
	return "", admindomain.AdminMaterialFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminMaterial(context.Context, string, admindomain.AdminMaterialForm) (admindomain.AdminMaterialFormPage, error) {
	return admindomain.AdminMaterialFormPage{}, nil
}

func (s *fakeAdminPanelService) ListAdminColors(context.Context) (admindomain.AdminColorListPage, error) {
	return admindomain.AdminColorListPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminColor() admindomain.AdminColorFormPage {
	return admindomain.AdminColorFormPage{}
}

func (s *fakeAdminPanelService) GetAdminColor(context.Context, string) (admindomain.AdminColorFormPage, error) {
	return admindomain.AdminColorFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminColor(context.Context, admindomain.AdminColorForm) (string, admindomain.AdminColorFormPage, error) {
	return "", admindomain.AdminColorFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminColor(context.Context, string, admindomain.AdminColorForm) (admindomain.AdminColorFormPage, error) {
	return admindomain.AdminColorFormPage{}, nil
}

func (s *fakeAdminPanelService) ListAdminBoxes(context.Context) (admindomain.AdminBoxListPage, error) {
	return admindomain.AdminBoxListPage{}, nil
}

func (s *fakeAdminPanelService) NewAdminBox() admindomain.AdminBoxFormPage {
	return admindomain.AdminBoxFormPage{}
}

func (s *fakeAdminPanelService) GetAdminBox(context.Context, string) (admindomain.AdminBoxFormPage, error) {
	return admindomain.AdminBoxFormPage{}, nil
}

func (s *fakeAdminPanelService) CreateAdminBox(context.Context, admindomain.AdminBoxForm) (string, admindomain.AdminBoxFormPage, error) {
	return "", admindomain.AdminBoxFormPage{}, nil
}

func (s *fakeAdminPanelService) UpdateAdminBox(context.Context, string, admindomain.AdminBoxForm) (admindomain.AdminBoxFormPage, error) {
	return admindomain.AdminBoxFormPage{}, nil
}

func (s *fakeAdminPanelService) WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	admindomain.NewTokenManager(admindomain.CookieOptions{}).WriteCookie(w, token, expiresAt)
}

func (s *fakeAdminPanelService) ClearCookie(w http.ResponseWriter) {
	admindomain.NewTokenManager(admindomain.CookieOptions{}).ClearCookie(w)
}

var errFakeAdminUnavailable = errors.New("admin fake unavailable")
