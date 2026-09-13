package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, authUserID string, tokenHash []byte, expiresAt time.Time) (Session, error)
	ResolveSession(ctx context.Context, tokenHash []byte) (Session, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}

type DashboardRepository interface {
	Dashboard(ctx context.Context) (Dashboard, error)
}

type OrderRepository interface {
	ListOrders(ctx context.Context, filter OrderListFilter) (OrderListPage, error)
	GetOrder(ctx context.Context, orderID string) (OrderDetail, error)
	ChangeProductionStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error
	ChangeShippingStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error
}

type CookieOptions struct {
	Secure bool
	Random io.Reader
}

type TokenManager struct {
	secure bool
	random io.Reader
}

func NewTokenManager(options CookieOptions) *TokenManager {
	random := options.Random
	if random == nil {
		random = rand.Reader
	}

	return &TokenManager{
		secure: options.Secure,
		random: random,
	}
}

func (m *TokenManager) GenerateToken() (string, error) {
	if m == nil {
		return "", ErrUnavailable
	}

	tokenBytes := make([]byte, TokenByteLength)
	if _, err := io.ReadFull(m.random, tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func (m *TokenManager) HashToken(token string) ([]byte, error) {
	if !ValidToken(token) {
		return nil, ErrInvalidToken
	}

	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}

func (m *TokenManager) ReadToken(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}

	cookie, err := r.Cookie(CookieName)
	if err != nil || !ValidToken(cookie.Value) {
		return "", false
	}

	return cookie.Value, true
}

func (m *TokenManager) Cookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/admin",
		Expires:  expiresAt,
		MaxAge:   int(SessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   m != nil && m.secure,
	}
}

func (m *TokenManager) WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, m.Cookie(token, expiresAt))
}

func (m *TokenManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/admin",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   m != nil && m.secure,
	})
}

func ValidToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != TokenByteLength {
		return false
	}

	return base64.RawURLEncoding.EncodeToString(decoded) == token
}

type Service struct {
	auth        AuthClient
	sessions    SessionRepository
	dashboard   DashboardRepository
	orders      OrderRepository
	catalog     CatalogRepository
	tokens      *TokenManager
	adminUserID string
	now         func() time.Time
}

func NewService(auth AuthClient, repository interface {
	SessionRepository
	DashboardRepository
	OrderRepository
}, adminUserID string, options CookieOptions) *Service {
	var catalog CatalogRepository
	if catalogRepository, ok := repository.(CatalogRepository); ok {
		catalog = catalogRepository
	}

	return &Service{
		auth:        auth,
		sessions:    repository,
		dashboard:   repository,
		orders:      repository,
		catalog:     catalog,
		tokens:      NewTokenManager(options),
		adminUserID: normalizeUUID(adminUserID),
		now:         time.Now,
	}
}

func (s *Service) Available() bool {
	return s != nil &&
		s.auth != nil &&
		s.sessions != nil &&
		s.dashboard != nil &&
		s.orders != nil &&
		s.tokens != nil &&
		ValidUUID(s.adminUserID)
}

func (s *Service) Login(ctx context.Context, email string, password string) (LoginResult, error) {
	if !s.Available() {
		return LoginResult{}, ErrUnavailable
	}
	if email == "" || password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	user, err := s.auth.SignInWithPassword(ctx, email, password)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if normalizeUUID(user.ID) != s.adminUserID {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := s.tokens.GenerateToken()
	if err != nil {
		return LoginResult{}, ErrUnavailable
	}
	tokenHash, err := s.tokens.HashToken(token)
	if err != nil {
		return LoginResult{}, ErrUnavailable
	}

	expiresAt := s.now().UTC().Add(SessionTTL)
	if _, err := s.sessions.CreateSession(ctx, s.adminUserID, tokenHash, expiresAt); err != nil {
		return LoginResult{}, ErrUnavailable
	}

	return LoginResult{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) ResolveSession(ctx context.Context, r *http.Request) (Session, error) {
	if !s.Available() {
		return Session{}, ErrUnavailable
	}

	token, ok := s.tokens.ReadToken(r)
	if !ok {
		return Session{}, ErrUnauthenticated
	}

	tokenHash, err := s.tokens.HashToken(token)
	if err != nil {
		return Session{}, ErrUnauthenticated
	}

	session, err := s.sessions.ResolveSession(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Session{}, ErrUnauthenticated
		}
		return Session{}, ErrUnavailable
	}

	if !session.ExpiresAt.After(s.now().UTC()) {
		_ = s.sessions.DeleteSession(ctx, tokenHash)
		return Session{}, ErrSessionExpired
	}
	if normalizeUUID(session.AuthUserID) != s.adminUserID {
		_ = s.sessions.DeleteSession(ctx, tokenHash)
		return Session{}, ErrUnauthenticated
	}

	return session, nil
}

func (s *Service) Logout(ctx context.Context, r *http.Request) error {
	if !s.Available() {
		return ErrUnavailable
	}

	token, ok := s.tokens.ReadToken(r)
	if !ok {
		return nil
	}

	tokenHash, err := s.tokens.HashToken(token)
	if err != nil {
		return nil
	}

	return s.sessions.DeleteSession(ctx, tokenHash)
}

func (s *Service) Dashboard(ctx context.Context) (Dashboard, error) {
	if !s.Available() {
		return Dashboard{}, ErrUnavailable
	}

	return s.dashboard.Dashboard(ctx)
}

func (s *Service) ListOrders(ctx context.Context, filter OrderListFilter) (OrderListPage, error) {
	if !s.Available() {
		return OrderListPage{}, ErrUnavailable
	}

	return s.orders.ListOrders(ctx, NormalizeOrderListFilter(filter))
}

func (s *Service) GetOrder(ctx context.Context, orderID string) (OrderDetail, error) {
	if !s.Available() {
		return OrderDetail{}, ErrUnavailable
	}
	orderID = normalizeUUID(orderID)
	if !ValidUUID(orderID) {
		return OrderDetail{}, ErrInvalidOrderID
	}

	return s.orders.GetOrder(ctx, orderID)
}

func (s *Service) ChangeProductionStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	orderID = normalizeUUID(orderID)
	actorAuthUserID = normalizeUUID(actorAuthUserID)
	targetStatus = strings.TrimSpace(targetStatus)
	if !ValidUUID(orderID) {
		return ErrInvalidOrderID
	}
	if !ValidUUID(actorAuthUserID) {
		return ErrInvalidTransition
	}

	return s.orders.ChangeProductionStatus(ctx, orderID, targetStatus, actorAuthUserID)
}

func (s *Service) ChangeShippingStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	orderID = normalizeUUID(orderID)
	actorAuthUserID = normalizeUUID(actorAuthUserID)
	targetStatus = strings.TrimSpace(targetStatus)
	if !ValidUUID(orderID) {
		return ErrInvalidOrderID
	}
	if !ValidUUID(actorAuthUserID) {
		return ErrInvalidTransition
	}

	return s.orders.ChangeShippingStatus(ctx, orderID, targetStatus, actorAuthUserID)
}

func (s *Service) WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	if s == nil || s.tokens == nil {
		return
	}
	s.tokens.WriteCookie(w, token, expiresAt)
}

func (s *Service) ClearCookie(w http.ResponseWriter) {
	if s == nil || s.tokens == nil {
		return
	}
	s.tokens.ClearCookie(w)
}
