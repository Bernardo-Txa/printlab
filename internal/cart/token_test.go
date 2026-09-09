package cart

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateTokenUsesExpectedEntropySize(t *testing.T) {
	manager := NewCookieManager(CookieOptions{})

	first, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token to be generated, got %v", err)
	}
	second, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected second token to be generated, got %v", err)
	}

	if first == second {
		t.Fatal("expected generated tokens to differ")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("expected token to be raw URL-safe base64, got %v", err)
	}

	if len(decoded) != TokenByteLength {
		t.Fatalf("expected %d random bytes, got %d", TokenByteLength, len(decoded))
	}
}

func TestHashTokenUsesSHA256WithoutReturningRawToken(t *testing.T) {
	manager := NewCookieManager(CookieOptions{})
	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token to be generated, got %v", err)
	}

	hash, err := manager.HashToken(token)
	if err != nil {
		t.Fatalf("expected token hash, got %v", err)
	}

	if len(hash) != HashByteLength {
		t.Fatalf("expected %d hash bytes, got %d", HashByteLength, len(hash))
	}

	if bytes.Equal([]byte(token), hash) {
		t.Fatal("expected raw token and hash to differ")
	}
}

func TestCartCookieAttributes(t *testing.T) {
	manager := NewCookieManager(CookieOptions{Secure: true})
	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("expected token to be generated, got %v", err)
	}
	expiresAt := time.Now().Add(TTL)

	cookie := manager.Cookie(token, expiresAt)

	if cookie.Name != CookieName {
		t.Fatalf("expected cookie name %q, got %q", CookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Fatal("expected cookie to contain raw token")
	}
	if cookie.Path != "/" {
		t.Fatalf("expected path /, got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}
	if cookie.MaxAge != int(TTL.Seconds()) {
		t.Fatalf("expected MaxAge for TTL, got %d", cookie.MaxAge)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Fatalf("expected Expires %s, got %s", expiresAt, cookie.Expires)
	}
	if cookie.Domain != "" {
		t.Fatalf("expected host-only cookie without Domain, got %q", cookie.Domain)
	}
}

func TestReadTokenRejectsInvalidCookie(t *testing.T) {
	manager := NewCookieManager(CookieOptions{})
	req := httptest.NewRequest(http.MethodGet, "/carrinho", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid"})

	if _, ok := manager.ReadToken(req); ok {
		t.Fatal("expected invalid cookie token to be ignored")
	}
}
