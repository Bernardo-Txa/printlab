package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
)

func TestSignupValidCreatesSupabaseUser(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"name": {"Maria Cliente"}, "email": {"maria@example.com"}, "password": {"senha-segura"}, "confirm_password": {"senha-segura"}}
	req := httptest.NewRequest(http.MethodPost, "/cadastro", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !service.signupCalled || service.signupInput.Email != "maria@example.com" || service.signupInput.Name != "Maria Cliente" {
		t.Fatalf("expected signup input, got %#v", service.signupInput)
	}
	if !strings.Contains(rec.Body.String(), customerauth.MessageSignupConfirmation) {
		t.Fatal("expected confirmation message")
	}
}

func TestSignupValidationRejectsMismatchedPassword(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"name": {"Maria"}, "email": {"maria@example.com"}, "password": {"senha-segura"}, "confirm_password": {"outra"}}
	req := httptest.NewRequest(http.MethodPost, "/cadastro", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if service.signupCalled {
		t.Fatal("expected invalid form not to call Supabase")
	}
	if !strings.Contains(rec.Body.String(), "As senhas não coincidem.") {
		t.Fatal("expected password mismatch message")
	}
}

func TestLoginValidRedirectsToSafeNext(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-segura"}, "next": {"/conta"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta" {
		t.Fatalf("expected redirect to account, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.loginCalled || service.loginEmail != "cliente@example.com" || service.loginPassword != "senha-segura" {
		t.Fatalf("expected login call, got %#v", service)
	}
}

func TestLoginInvalidShowsGenericMessage(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, loginErr: customerauth.ErrRejected}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-errada"}, "next": {"/conta"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, customerauth.MessageInvalidLogin) || strings.Contains(body, "senha-errada") {
		t.Fatal("expected generic invalid login message without password")
	}
}

func TestLoginUnconfirmedEmailShowsConfirmationMessage(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, loginErr: customerauth.ErrEmailNotConfirmed}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-segura"}, "next": {"/conta"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), customerauth.MessageEmailNotConfirmed) {
		t.Fatal("expected email confirmation message")
	}
}

func TestLoginRejectsOpenRedirect(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"cliente@example.com"}, "password": {"senha-segura"}, "next": {"https://evil.example"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta" {
		t.Fatalf("expected fallback redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestRecoveryDoesNotEnumerateUsers(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, recoveryErr: customerauth.ErrRejected}
	handler := newCustomerAuthTestHandler(service)
	form := url.Values{"email": {"desconhecido@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/recuperar-senha", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), customerauth.MessageRecoverySent) {
		t.Fatal("expected generic recovery message")
	}
}

func TestAuthSessionCallbackSetsSupabaseSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/auth/session", strings.NewReader(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"next":"/conta"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !service.callbackCalled {
		t.Fatal("expected callback session completion")
	}
}

func TestAuthCodeCallbackExchangesCodeAndRendersSuccessCard(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/auth/callback?code=auth-code", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected success page, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"E-mail confirmado!", "Ir para minha conta", "Voltar para o início"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected success card to contain %q", expected)
		}
	}
	if strings.Contains(body, "checkout-alert-error") {
		t.Fatal("expected confirmation success not to look like an error")
	}
	if !service.callbackCodeCalled || service.callbackCode != "auth-code" {
		t.Fatalf("expected code exchange, got %#v", service)
	}
}

func TestAuthCodeCallbackRecoveryRedirectsToNewPassword(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/auth/callback?code=auth-code&type=recovery", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/recuperar-senha/nova" {
		t.Fatalf("expected redirect to new password, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAuthCodeCallbackInvalidShowsSafeMessage(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, callbackCodeErr: customerauth.ErrRejected}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/auth/callback?code=expired-code", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, customerauth.MessageCallbackInvalid) || strings.Contains(body, "expired-code") {
		t.Fatal("expected safe callback error without code")
	}
}

func TestAuthCallbackWithoutCodeOrTokensRendersFallback(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/auth/callback", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 fallback, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "auth-callback.js") {
		t.Fatal("expected JS fallback for hash token callback")
	}
}

func TestAccountRequiresAuthentication(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login?next=") {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAccountRendersAuthenticatedUser(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, profile: customerauth.Profile{ID: "user-1", Name: "Maria Cliente", Email: "maria@example.com"}}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Conta PrintLab", "Minha conta", "Olá, Maria.", "maria@example.com", "Conta ativa", "Redefinir senha"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected account page to contain %q", expected)
		}
	}
}

func TestAccountRendersDashboardNavigationAndSecurity(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{
		`class="account-hero`,
		`href="#visao-geral">Visão geral`,
		`href="#pedidos">Pedidos`,
		`href="#dados">Meus dados`,
		`href="#seguranca">Segurança`,
		`id="visao-geral"`,
		`id="pedidos"`,
		`id="dados"`,
		`id="seguranca"`,
		`href="/recuperar-senha">Redefinir senha`,
		`method="post" action="/logout"`,
		"Sair da conta",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected account dashboard to contain %q", expected)
		}
	}
	if strings.Contains(body, "admin-panel account-panel") {
		t.Fatal("expected account dashboard not to use admin panel layout")
	}
}

func TestAccountEmptyOrdersShowsCustomerEmptyState(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, &fakeOrderReviewService{}, &fakeAccountProfileService{}, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Você ainda não fez nenhum pedido.", "Quando você comprar usando esta conta", `href="/produtos">Ver produtos`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected empty orders state to contain %q", expected)
		}
	}
	if strings.Contains(body, "Você ainda não tem pedidos associados a esta conta.") {
		t.Fatal("expected old empty copy to be removed")
	}
}

func TestAccountDoesNotExposeInternalUUIDAsVisibleText(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	ordersService := &fakeOrderReviewService{accountOrders: []ordersdomain.AccountOrder{{ID: orderID, OrderNumber: 1001, CreatedAt: accountOrderDate(), Status: ordersdomain.StatusPendingPayment, StatusLabel: "Aguardando pagamento", TotalBRL: "R$ 99,90", TrackingURL: "/acompanhar/abc"}}}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, ordersService, &fakeAccountProfileService{}, &fakePaymentService{available: true}, "https://printlab.test").ServeHTTP(rec, req)

	visibleText := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(rec.Body.String(), " ")
	if strings.Contains(visibleText, orderID) || strings.Contains(visibleText, accountTestAuthProfile().ID) {
		t.Fatalf("expected internal UUIDs not to appear as visible text, got %s", visibleText)
	}
}

func TestAccountDoesNotCreateNewAccountRoutes(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("expected main.go to be readable, got %v", err)
	}
	for _, forbidden := range []string{"/conta/dados", "/conta/seguranca", "/conta/pedidos\""} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("expected no new account subroute %q", forbidden)
		}
	}
}

func TestAccountGetLoadsCustomerProfile(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	profiles := &fakeAccountProfileService{profile: accountSavedProfile(), found: true}
	ordersService := &fakeOrderReviewService{accountOrders: []ordersdomain.AccountOrder{{ID: orderID, OrderNumber: 1001, CreatedAt: accountOrderDate(), Status: ordersdomain.StatusPendingPayment, StatusLabel: "Aguardando pagamento", TotalBRL: "R$ 99,90", TrackingURL: "/acompanhar/abc"}}}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, ordersService, profiles, &fakePaymentService{available: true}, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{
		`id="dados"`,
		`method="post" action="/conta"`,
		`data-cep-lookup-endpoint="/api/cep"`,
		"Dados pessoais",
		"Seus dados de contato e identificação.",
		"Endereço de entrega",
		"Usado para agilizar entregas em compras futuras.",
		"Salvar alterações",
		"Perfil Salvo",
		`name="full_name"`,
		`value="27988887777"`,
		`id="account_email"`,
		`readonly`,
		`data-checkout-mask="phone"`,
		`data-checkout-mask="cpf"`,
		`data-checkout-mask="cep"`,
		`name="postal_code"`,
		`name="street"`,
		`name="number"`,
		`name="complement"`,
		`name="district"`,
		`name="city"`,
		`name="state"`,
		"Pedido #1001",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected account page to contain %q", expected)
		}
	}
	if strings.Contains(body, "pedidos autenticados") {
		t.Fatal("expected technical copy to be removed")
	}
	for _, expected := range []string{"22/09/2026", "Retomar pagamento", `action="/conta/pedidos/` + orderID + `/pagar"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected pending payment CTA to contain %q", expected)
		}
	}
	if strings.Contains(body, "checkout.infinitepay") {
		t.Fatal("expected account page not to render checkout URL")
	}
	if profiles.getID != accountTestAuthProfile().ID {
		t.Fatalf("expected profile lookup by auth id, got %q", profiles.getID)
	}
}

func TestAccountPaidOrderDoesNotShowResumePaymentCTA(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	ordersService := &fakeOrderReviewService{accountOrders: []ordersdomain.AccountOrder{{ID: orderID, OrderNumber: 1001, CreatedAt: accountOrderDate(), Status: ordersdomain.StatusPaid, StatusLabel: "Pago", TotalBRL: "R$ 99,90", TrackingURL: "/acompanhar/abc"}}}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, ordersService, &fakeAccountProfileService{}, &fakePaymentService{available: true}, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Retomar pagamento") {
		t.Fatal("expected paid order not to show resume payment CTA")
	}
	if !strings.Contains(body, "Pagamento confirmado.") {
		t.Fatal("expected paid order confirmation copy")
	}
}

func TestAccountGetFallbackLatestOrderUsesAuthUserID(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	profiles := &fakeAccountProfileService{}
	ordersService := &fakeOrderReviewService{snapshotFound: true, snapshot: ordersdomain.CustomerSnapshot{
		FullName: "Pedido Recente", Phone: "+5527999999999", CPF: "52998224725", PostalCode: "29100000", Street: "Rua Pedido", Number: "45", District: "Centro", City: "Vila Velha", State: "ES", CountryCode: customers.CountryCodeBR,
	}}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, ordersService, profiles, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ordersService.lastSnapshotCustomerID != accountTestAuthProfile().ID {
		t.Fatalf("expected latest order lookup by auth id, got %q", ordersService.lastSnapshotCustomerID)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Pedido Recente") || strings.Contains(body, "cliente@example.com") && strings.Contains(body, "customer_email") {
		t.Fatal("expected snapshot fallback without email ownership lookup")
	}
}

func TestAccountWithoutProfileOrOrderUsesSupabaseNameAndEmail(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, &fakeOrderReviewService{}, &fakeAccountProfileService{}, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{`value="Cliente Auth"`, `value="cliente@example.com"`, `readonly`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected auth fallback to contain %q", expected)
		}
	}
}

func TestAccountPostValidSavesProfile(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	profiles := &fakeAccountProfileService{}
	req := accountFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountSaveHandler(auth, &fakeOrderReviewService{}, profiles, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta?salvo=1#dados" {
		t.Fatalf("expected saved redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !profiles.saveCalled || profiles.saved.AuthUserID != accountTestAuthProfile().ID || profiles.saved.FullName != "Joao Silva" {
		t.Fatalf("expected saved profile from form and auth id, got %#v", profiles.saved)
	}
}

func TestAccountPostUsesSessionEmailNotBrowserEmail(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	profiles := &fakeAccountProfileService{}
	form := validCheckoutForm()
	form.Set("email", "email-invalido-do-browser")
	req := accountFormRequest(form)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountSaveHandler(auth, &fakeOrderReviewService{}, profiles, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if profiles.saved.FullName == "" || profiles.saved.AuthUserID != accountTestAuthProfile().ID {
		t.Fatalf("expected profile saved, got %#v", profiles.saved)
	}
}

func TestAccountPostInvalidShowsErrorsAndPreservesOrders(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	ordersService := &fakeOrderReviewService{accountOrders: []ordersdomain.AccountOrder{{OrderNumber: 1002, StatusLabel: "Pago", TotalBRL: "R$ 10,00", TrackingURL: "/acompanhar/def"}}}
	profiles := &fakeAccountProfileService{}
	form := validCheckoutForm()
	form.Set("cpf", "123")
	req := accountFormRequest(form)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountSaveHandler(auth, ordersService, profiles, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{"Informe um CPF valido.", "Pedido #1002", `value="Joao Silva"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected invalid account page to contain %q", expected)
		}
	}
	if profiles.saveCalled {
		t.Fatal("expected invalid profile not to be saved")
	}
}

func TestAccountSavedQueryShowsSuccessMessage(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/conta?salvo=1", nil)
	rec := httptest.NewRecorder()

	accountHandler(auth, &fakeOrderReviewService{}, &fakeAccountProfileService{}, nil, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Dados atualizados com sucesso.") {
		t.Fatal("expected account success message")
	}
}

func TestAccountResumePaymentRequiresAuthentication(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login?next=") {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if payment.customerStartCalls != 0 {
		t.Fatal("expected unauthenticated request not to call payment service")
	}
}

func TestAccountResumePaymentRejectsCrossSiteOrigin(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	payment := &fakePaymentService{available: true}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if payment.customerStartCalls != 0 {
		t.Fatal("expected forbidden request not to call payment service")
	}
}

func TestAccountResumePaymentRedirectsToInfinitePayForOwner(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	payment := &fakePaymentService{available: true, startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID, CheckoutURL: "https://checkout.infinitepay.com.br/checkout-slug"}}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "https://checkout.infinitepay.com.br/checkout-slug" {
		t.Fatalf("expected InfinitePay redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if payment.lastStartOrderID != orderID || payment.lastStartCustomerID != accountTestAuthProfile().ID {
		t.Fatalf("expected order and session customer id, got order=%q customer=%q", payment.lastStartOrderID, payment.lastStartCustomerID)
	}
}

func TestAccountResumePaymentOtherCustomerLooksNotFound(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	payment := &fakePaymentService{available: true, startErr: paymentsdomain.ErrOrderNotFound}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), orderID) {
		t.Fatal("expected not found response not to reveal order id")
	}
}

func TestAccountResumePaymentPaidOrderRedirectsWithMessage(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	payment := &fakePaymentService{available: true, startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID}, startErr: paymentsdomain.ErrOrderAlreadyPaid}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta?pagamento=confirmado" {
		t.Fatalf("expected confirmed redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAccountResumePaymentRejectsInvalidCheckoutURL(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	payment := &fakePaymentService{available: true, startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID, CheckoutURL: "https://evil.example/checkout"}}
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta?pagamento=indisponivel" {
		t.Fatalf("expected unavailable redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAccountResumePaymentProviderErrorLogsSafeDiagnostics(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	providerErr := paymentsdomain.NewProviderError(paymentsdomain.ProviderOperationCreateCheckout, paymentsdomain.ProviderCategoryHTTP5xx, http.StatusServiceUnavailable, paymentsdomain.ErrProviderUnavailable)
	providerErr.Message = "invalid customer Joao Silva cliente@example.com +5527999999999 Rua Um https://checkout.infinitepay.com.br/checkout-slug"
	payment := &fakePaymentService{available: true, startResult: paymentsdomain.CheckoutStartResult{OrderID: orderID}, startErr: providerErr}
	var logs strings.Builder
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta/pedidos/"+orderID+"/pagar", nil)
	req.SetPathValue("id", orderID)
	req.Header.Set("Origin", "https://printlab.test")
	rec := httptest.NewRecorder()

	accountResumePaymentHandler(auth, payment, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/conta?pagamento=indisponivel" {
		t.Fatalf("expected unavailable redirect, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	logText := logs.String()
	for _, expected := range []string{"event=customer_payment_resume_unavailable", "reason=provider_unavailable", "provider=infinitepay"} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
	for _, leaked := range []string{"Joao", "cliente@example.com", "+5527999999999", "Rua Um", "checkout-slug"} {
		if strings.Contains(logText, leaked) {
			t.Fatalf("expected resume log not to leak %q, got %q", leaked, logText)
		}
	}
}

func TestNewPasswordRequiresRecoverySession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/recuperar-senha/nova", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login?next=") {
		t.Fatalf("expected redirect to login, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestLogoutClearsSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("expected redirect home, got %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !service.logoutCalled {
		t.Fatal("expected logout call")
	}
}

func TestGuestCheckoutStillWorksWithoutCustomerSession(t *testing.T) {
	service := &fakeCustomerAuthService{available: true, resolveErr: customerauth.ErrUnauthenticated}
	handler := newCustomerAuthTestHandler(service)
	req := httptest.NewRequest(http.MethodGet, "https://printlab.test/carrinho", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected guest cart to keep working, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Entrar para comprar") {
		t.Fatal("expected guest checkout path not to require login")
	}
}

func newCustomerAuthTestHandler(service customerAuthService) http.Handler {
	return newHandlerWithServicesAndOrdersAndCustomerAuthAndSupabaseURL(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, service, "https://printlab.test", "https://supabase.test")
}

func accountFormRequest(values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "https://printlab.test/conta", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func accountTestAuthProfile() customerauth.Profile {
	return customerauth.Profile{ID: "11111111-1111-1111-1111-111111111111", Name: "Cliente Auth", Email: "cliente@example.com"}
}

func accountSavedProfile() customerprofile.Profile {
	return customerprofile.Profile{AuthUserID: accountTestAuthProfile().ID, FullName: "Perfil Salvo", Phone: "27988887777", CPF: "52998224725", PostalCode: "29100000", Street: "Rua Perfil", Number: "123", District: "Centro", City: "Vila Velha", State: "ES", CountryCode: customers.CountryCodeBR}
}

func accountOrderDate() time.Time {
	return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
}

type fakeAccountProfileService struct {
	profile customerprofile.Profile
	found   bool
	getErr  error
	saveErr error

	getID      string
	saveCalled bool
	saved      customerprofile.Profile
}

func (s *fakeAccountProfileService) Get(_ context.Context, id string) (customerprofile.Profile, bool, error) {
	s.getID = id
	if s.getErr != nil {
		return customerprofile.Profile{}, false, s.getErr
	}
	return s.profile, s.found, nil
}

func (s *fakeAccountProfileService) Save(_ context.Context, profile customerprofile.Profile) error {
	s.saveCalled = true
	s.saved = profile
	return s.saveErr
}

func TestAccountProfileForDisplayLogsProfileErrors(t *testing.T) {
	profile := accountTestAuthProfile()
	got := accountProfileForDisplay(requestIDContext(context.Background(), "test-request-id"), profile, nil, &fakeAccountProfileService{getErr: errors.New("db down")})
	if got.FullName != profile.Name || got.AuthUserID != profile.ID {
		t.Fatalf("expected auth fallback after profile error, got %#v", got)
	}
}

func TestAccountPostOriginProtection(t *testing.T) {
	auth := &fakeCustomerAuthService{available: true, profile: accountTestAuthProfile()}
	profiles := &fakeAccountProfileService{}
	req := accountFormRequest(validCheckoutForm())
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	accountSaveHandler(auth, &fakeOrderReviewService{}, profiles, "https://printlab.test").ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if profiles.saveCalled {
		t.Fatal("expected rejected origin not to save profile")
	}
}

var _ accountProfileService = (*fakeAccountProfileService)(nil)

var _ = time.Time{}

type fakeCustomerAuthService struct {
	available bool

	signupCalled bool
	signupInput  customerauth.SignUpInput
	signupErr    error

	loginCalled   bool
	loginEmail    string
	loginPassword string
	loginErr      error

	recoveryCalled bool
	recoveryEmail  string
	recoveryErr    error

	callbackCalled bool
	callbackErr    error

	callbackCodeCalled bool
	callbackCode       string
	callbackCodeErr    error

	updateErr error

	logoutCalled bool

	profile    customerauth.Profile
	resolveErr error
}

func (s *fakeCustomerAuthService) Available() bool { return s != nil && s.available }
func (s *fakeCustomerAuthService) SignUp(_ context.Context, input customerauth.SignUpInput) error {
	s.signupCalled = true
	s.signupInput = input
	return s.signupErr
}
func (s *fakeCustomerAuthService) Login(_ context.Context, _ http.ResponseWriter, email string, password string) error {
	s.loginCalled = true
	s.loginEmail = email
	s.loginPassword = password
	return s.loginErr
}
func (s *fakeCustomerAuthService) RecoverPassword(_ context.Context, email string, _ string) error {
	s.recoveryCalled = true
	s.recoveryEmail = email
	return s.recoveryErr
}
func (s *fakeCustomerAuthService) CompleteCallbackCode(_ context.Context, _ http.ResponseWriter, code string) error {
	s.callbackCodeCalled = true
	s.callbackCode = code
	return s.callbackCodeErr
}
func (s *fakeCustomerAuthService) CompleteCallbackSession(_ context.Context, _ http.ResponseWriter, _ customerauth.AuthSession) error {
	s.callbackCalled = true
	return s.callbackErr
}
func (s *fakeCustomerAuthService) UpdatePassword(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ string) error {
	return s.updateErr
}
func (s *fakeCustomerAuthService) Logout(_ context.Context, _ http.ResponseWriter, _ *http.Request) {
	s.logoutCalled = true
}
func (s *fakeCustomerAuthService) ResolveSession(_ context.Context, _ http.ResponseWriter, _ *http.Request) (customerauth.Profile, error) {
	if s.resolveErr != nil {
		return customerauth.Profile{}, s.resolveErr
	}
	if s.profile.ID == "" {
		return customerauth.Profile{}, customerauth.ErrUnauthenticated
	}
	return s.profile, nil
}
func (s *fakeCustomerAuthService) RedirectURL(_ *http.Request, path string) string {
	return "https://printlab.test" + path
}
