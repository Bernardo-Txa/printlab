package admin

import (
	"errors"
	"time"
)

const (
	CookieName      = "printlab_admin_session"
	TokenByteLength = 32
	HashByteLength  = 32
	SessionTTL      = 8 * time.Hour

	InvalidCredentialsMessage = "E-mail ou senha inválidos."
)

var (
	ErrAuthRejected       = errors.New("admin auth rejected")
	ErrAuthUnavailable    = errors.New("admin auth unavailable")
	ErrInvalidCredentials = errors.New("admin invalid credentials")
	ErrUnauthenticated    = errors.New("admin unauthenticated")
	ErrSessionNotFound    = errors.New("admin session not found")
	ErrSessionExpired     = errors.New("admin session expired")
	ErrInvalidToken       = errors.New("admin invalid session token")
	ErrUnavailable        = errors.New("admin unavailable")
)

type AuthUser struct {
	ID string
}

type Session struct {
	ID         string
	AuthUserID string
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
}

type Dashboard struct {
	PendingPayment        int
	PaidWaitingProduction int
	InProduction          int
	WaitingShipment       int
}

type OrderStatusSnapshot struct {
	OrderStatus      string
	ProductionStatus string
	ShippingStatus   string
}
