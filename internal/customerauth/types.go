package customerauth

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	AccessCookieName  = "printlab_customer_access_token"
	RefreshCookieName = "printlab_customer_refresh_token"
	DefaultAccessTTL  = time.Hour
	RefreshCookieTTL  = 30 * 24 * time.Hour

	MessageSignupConfirmation = "Conta criada com sucesso! Enviamos um e-mail para confirmar seu endereço. Abra o e-mail e clique no link de confirmação antes de entrar. Não recebeu? Verifique o spam ou solicite um novo e-mail."
	MessageInvalidLogin       = "E-mail ou senha inválidos."
	MessageEmailNotConfirmed  = "Confirme seu e-mail antes de entrar. Verifique sua caixa de entrada."
	MessageCallbackInvalid    = "Este link expirou ou já foi utilizado. Solicite um novo link."
	MessageRecoverySent       = "Se existir uma conta para este e-mail, enviaremos as instruções de recuperação."
	MessagePasswordUpdated    = "Senha atualizada. Entre novamente para continuar."
)

var (
	ErrUnavailable         = errors.New("customer auth unavailable")
	ErrConfiguration       = errors.New("customer auth configuration invalid")
	ErrRejected            = errors.New("customer auth rejected")
	ErrInvalidResponse     = errors.New("customer auth invalid response")
	ErrRateLimited         = errors.New("customer auth rate limited")
	ErrUnauthenticated     = errors.New("customer unauthenticated")
	ErrValidation          = errors.New("customer auth validation failed")
	ErrInvalidRedirect     = errors.New("customer auth invalid redirect")
	ErrExpiredSession      = errors.New("customer session expired")
	ErrInvalidSessionToken = errors.New("customer invalid session token")
	ErrEmailNotConfirmed   = errors.New("customer email not confirmed")
)

type Provider interface {
	SignUp(context.Context, SignUpInput) (SignUpResult, error)
	SignInWithPassword(context.Context, string, string) (AuthSession, error)
	RecoverPassword(context.Context, string, string) error
	ExchangeCode(context.Context, string) (AuthSession, error)
	UpdatePassword(context.Context, string, string) error
	RefreshSession(context.Context, string) (AuthSession, error)
	GetUser(context.Context, string) (AuthUser, error)
	Logout(context.Context, string) error
}

type SignUpInput struct {
	Name        string
	Email       string
	Password    string
	RedirectURL string
}

type SignUpResult struct {
	User AuthUser
}

type AuthSession struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         AuthUser `json:"user"`
}

type AuthUser struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	EmailConfirmedAt *time.Time     `json:"email_confirmed_at"`
	UserMetadata     map[string]any `json:"user_metadata"`
}

func (u AuthUser) FullName() string {
	for _, key := range []string{"full_name", "name"} {
		if value, ok := u.UserMetadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type Profile struct {
	ID    string
	Name  string
	Email string
}

type SignupForm struct {
	Name            string
	Email           string
	Password        string
	ConfirmPassword string
	Errors          FieldErrors
	Message         string
}

type LoginForm struct {
	Email   string
	Next    string
	Errors  FieldErrors
	Message string
}

type RecoveryForm struct {
	Email   string
	Errors  FieldErrors
	Message string
}

type NewPasswordForm struct {
	Errors  FieldErrors
	Message string
}

type FieldErrors map[string]string

func (e FieldErrors) Any() bool {
	return len(e) > 0
}

func (e FieldErrors) Message(field string) string {
	if e == nil {
		return ""
	}
	return e[field]
}
