package cart

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

type CookieOptions struct {
	Secure bool
	Random io.Reader
}

type CookieManager struct {
	secure bool
	random io.Reader
}

func NewCookieManager(options CookieOptions) *CookieManager {
	random := options.Random
	if random == nil {
		random = rand.Reader
	}

	return &CookieManager{
		secure: options.Secure,
		random: random,
	}
}

func (m *CookieManager) GenerateToken() (string, error) {
	if m == nil {
		return "", ErrUnavailable
	}

	tokenBytes := make([]byte, TokenByteLength)
	if _, err := io.ReadFull(m.random, tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func (m *CookieManager) ReadToken(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}

	cookie, err := r.Cookie(CookieName)
	if err != nil || !ValidToken(cookie.Value) {
		return "", false
	}

	return cookie.Value, true
}

func (m *CookieManager) HashToken(token string) ([]byte, error) {
	if !ValidToken(token) {
		return nil, ErrInvalidToken
	}

	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}

func (m *CookieManager) Cookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(TTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m != nil && m.secure,
	}
}

func (m *CookieManager) SetCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, m.Cookie(token, expiresAt))
}

func (m *CookieManager) ExpireCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
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
