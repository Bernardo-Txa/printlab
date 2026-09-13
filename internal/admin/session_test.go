package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAdminTokenUsesExpectedEntropySize(t *testing.T) {
	manager := NewTokenManager(CookieOptions{})

	first, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token, got %v", err)
	}
	second, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected second token, got %v", err)
	}
	if first == second {
		t.Fatal("expected generated tokens to differ")
	}
	if !ValidToken(first) {
		t.Fatal("expected generated token to be valid")
	}
}

func TestAdminHashTokenUsesSHA256WithoutReturningRawToken(t *testing.T) {
	manager := NewTokenManager(CookieOptions{})
	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token, got %v", err)
	}

	hash, err := manager.HashToken(token)
	if err != nil {
		t.Fatalf("expected hash, got %v", err)
	}
	if len(hash) != HashByteLength {
		t.Fatalf("expected hash size %d, got %d", HashByteLength, len(hash))
	}
	if bytes.Equal([]byte(token), hash) {
		t.Fatal("expected token hash to differ from raw token")
	}
}

func TestAdminCookieAttributes(t *testing.T) {
	manager := NewTokenManager(CookieOptions{Secure: true})
	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token, got %v", err)
	}
	expiresAt := time.Now().Add(SessionTTL)

	cookie := manager.Cookie(token, expiresAt)
	if cookie.Name != CookieName {
		t.Fatalf("expected cookie name %q, got %q", CookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Fatal("expected cookie to contain raw token")
	}
	if cookie.Path != "/admin" {
		t.Fatalf("expected admin path, got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected SameSite=Strict, got %v", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}
	if cookie.MaxAge != int(SessionTTL.Seconds()) {
		t.Fatalf("expected MaxAge for session TTL, got %d", cookie.MaxAge)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Fatalf("expected Expires %s, got %s", expiresAt, cookie.Expires)
	}
	if cookie.Domain != "" {
		t.Fatalf("expected host-only cookie without Domain, got %q", cookie.Domain)
	}
}

func TestAdminCookieCleared(t *testing.T) {
	manager := NewTokenManager(CookieOptions{Secure: true})
	rec := httptest.NewRecorder()

	manager.ClearCookie(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %#v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != CookieName || cookie.Value != "" || cookie.Path != "/admin" || cookie.MaxAge != -1 {
		t.Fatalf("expected cleared admin cookie, got %#v", cookie)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || !cookie.Secure {
		t.Fatalf("expected secure cleared cookie attributes, got %#v", cookie)
	}
}

func TestAdminServiceCreatesAndResolvesAuthorizedSession(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, now)

	result, err := service.Login(context.Background(), "admin@example.com", "correct-password")
	if err != nil {
		t.Fatalf("expected login, got %v", err)
	}
	if !result.ExpiresAt.Equal(now.Add(SessionTTL)) {
		t.Fatalf("expected 8h TTL, got %s", result.ExpiresAt.Sub(now))
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(service.tokens.Cookie(result.Token, result.ExpiresAt))

	session, err := service.ResolveSession(context.Background(), req)
	if err != nil {
		t.Fatalf("expected valid session, got %v", err)
	}
	if session.AuthUserID != testAdminUserID {
		t.Fatalf("expected admin user id, got %q", session.AuthUserID)
	}
}

func TestAdminServiceRejectsUnknownSession(t *testing.T) {
	service := newTestService(newFakeRepository(), testAdminUserID, time.Now())
	token, err := service.tokens.GenerateToken()
	if err != nil {
		t.Fatalf("expected token, got %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(service.tokens.Cookie(token, time.Now().Add(SessionTTL)))

	_, err = service.ResolveSession(context.Background(), req)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestAdminServiceRejectsExpiredSessionAndDeletesIt(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, now)
	token, hash := mustTokenAndHash(t, service.tokens)
	repo.sessions[string(hash)] = Session{
		ID:         "session-id",
		AuthUserID: testAdminUserID,
		TokenHash:  hash,
		CreatedAt:  now.Add(-9 * time.Hour),
		ExpiresAt:  now.Add(-time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(service.tokens.Cookie(token, now.Add(SessionTTL)))

	_, err := service.ResolveSession(context.Background(), req)
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected expired session, got %v", err)
	}
	if _, ok := repo.sessions[string(hash)]; ok {
		t.Fatal("expected expired session to be deleted")
	}
}

func TestAdminServiceRejectsSessionFromAnotherUser(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, now)
	token, hash := mustTokenAndHash(t, service.tokens)
	repo.sessions[string(hash)] = Session{
		ID:         "session-id",
		AuthUserID: "22222222-2222-2222-2222-222222222222",
		TokenHash:  hash,
		CreatedAt:  now,
		ExpiresAt:  now.Add(SessionTTL),
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(service.tokens.Cookie(token, now.Add(SessionTTL)))

	_, err := service.ResolveSession(context.Background(), req)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if _, ok := repo.sessions[string(hash)]; ok {
		t.Fatal("expected unauthorized session to be deleted")
	}
}

func TestAdminServiceDoesNotCreateSessionForNonAdminUser(t *testing.T) {
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, time.Now())
	service.auth = fakeAuthClient{userID: "22222222-2222-2222-2222-222222222222"}

	_, err := service.Login(context.Background(), "other@example.com", "correct-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("expected no session to be created, got %#v", repo.sessions)
	}
}

func TestAdminServiceLogoutDeletesSession(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, now)
	result, err := service.Login(context.Background(), "admin@example.com", "correct-password")
	if err != nil {
		t.Fatalf("expected login, got %v", err)
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(repo.sessions))
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.AddCookie(service.tokens.Cookie(result.Token, result.ExpiresAt))

	if err := service.Logout(context.Background(), req); err != nil {
		t.Fatalf("expected logout, got %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("expected session to be deleted, got %#v", repo.sessions)
	}
}

func TestDashboardFromOrderStatuses(t *testing.T) {
	dashboard := DashboardFromOrderStatuses([]OrderStatusSnapshot{
		{OrderStatus: OrderStatusPendingPayment, ProductionStatus: ProductionStatusWaiting, ShippingStatus: ShippingStatusWaiting},
		{OrderStatus: OrderStatusPaid, ProductionStatus: ProductionStatusWaiting, ShippingStatus: ShippingStatusWaiting},
		{OrderStatus: OrderStatusPaid, ProductionStatus: ProductionStatusInProduction, ShippingStatus: ShippingStatusWaiting},
		{OrderStatus: OrderStatusPaid, ProductionStatus: ProductionStatusCompleted, ShippingStatus: ShippingStatusWaiting},
		{OrderStatus: OrderStatusPaid, ProductionStatus: ProductionStatusCompleted, ShippingStatus: "shipped"},
	})

	if dashboard.PendingPayment != 1 ||
		dashboard.PaidWaitingProduction != 1 ||
		dashboard.InProduction != 1 ||
		dashboard.WaitingShipment != 1 {
		t.Fatalf("unexpected dashboard counts: %#v", dashboard)
	}
}

func newTestService(repo *fakeRepository, adminUserID string, now time.Time) *Service {
	service := NewService(fakeAuthClient{userID: adminUserID}, repo, adminUserID, CookieOptions{})
	service.now = func() time.Time { return now }
	return service
}

func mustTokenAndHash(t *testing.T, manager *TokenManager) (string, []byte) {
	t.Helper()

	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token, got %v", err)
	}
	hash, err := manager.HashToken(token)
	if err != nil {
		t.Fatalf("expected hash, got %v", err)
	}
	return token, hash
}

type fakeAuthClient struct {
	userID string
	err    error
}

func (c fakeAuthClient) SignInWithPassword(context.Context, string, string) (AuthUser, error) {
	if c.err != nil {
		return AuthUser{}, c.err
	}
	return AuthUser{ID: c.userID}, nil
}

type fakeRepository struct {
	sessions  map[string]Session
	dashboard Dashboard
	orders    OrderListPage
	order     OrderDetail
	err       error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{sessions: map[string]Session{}}
}

func (r *fakeRepository) CreateSession(_ context.Context, authUserID string, tokenHash []byte, expiresAt time.Time) (Session, error) {
	if r.err != nil {
		return Session{}, r.err
	}
	session := Session{
		ID:         "session-id",
		AuthUserID: authUserID,
		TokenHash:  tokenHash,
		CreatedAt:  expiresAt.Add(-SessionTTL),
		ExpiresAt:  expiresAt,
	}
	r.sessions[string(tokenHash)] = session
	return session, nil
}

func (r *fakeRepository) ResolveSession(_ context.Context, tokenHash []byte) (Session, error) {
	if r.err != nil {
		return Session{}, r.err
	}
	session, ok := r.sessions[string(tokenHash)]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (r *fakeRepository) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(r.sessions, string(tokenHash))
	return nil
}

func (r *fakeRepository) Dashboard(context.Context) (Dashboard, error) {
	if r.err != nil {
		return Dashboard{}, r.err
	}
	return r.dashboard, nil
}

func (r *fakeRepository) ListOrders(context.Context, OrderListFilter) (OrderListPage, error) {
	if r.err != nil {
		return OrderListPage{}, r.err
	}
	return r.orders, nil
}

func (r *fakeRepository) GetOrder(context.Context, string) (OrderDetail, error) {
	if r.err != nil {
		return OrderDetail{}, r.err
	}
	return r.order, nil
}

func (r *fakeRepository) ChangeProductionStatus(context.Context, string, string, string) error {
	return r.err
}

func (r *fakeRepository) ChangeShippingStatus(context.Context, string, string, string) error {
	return r.err
}
