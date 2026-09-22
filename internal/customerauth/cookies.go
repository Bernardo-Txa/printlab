package customerauth

import (
	"net/http"
	"strings"
	"time"
)

type CookieOptions struct {
	Secure bool
}

type CookieManager struct {
	secure bool
}

func NewCookieManager(options CookieOptions) *CookieManager {
	return &CookieManager{secure: options.Secure}
}

func (m *CookieManager) ReadAccessToken(r *http.Request) (string, bool) {
	return readCookieValue(r, AccessCookieName)
}

func (m *CookieManager) ReadRefreshToken(r *http.Request) (string, bool) {
	return readCookieValue(r, RefreshCookieName)
}

func readCookieValue(r *http.Request, name string) (string, bool) {
	if r == nil {
		return "", false
	}
	cookie, err := r.Cookie(name)
	if err != nil || strings.TrimSpace(cookie.Value) == "" || strings.ContainsAny(cookie.Value, "\r\n") {
		return "", false
	}
	return cookie.Value, true
}

func (m *CookieManager) WriteSession(w http.ResponseWriter, session AuthSession, now time.Time) {
	accessTTL := DefaultAccessTTL
	if session.ExpiresIn > 0 {
		accessTTL = time.Duration(session.ExpiresIn) * time.Second
	}
	m.writeCookie(w, AccessCookieName, session.AccessToken, now.Add(accessTTL), int(accessTTL.Seconds()))
	m.writeCookie(w, RefreshCookieName, session.RefreshToken, now.Add(RefreshCookieTTL), int(RefreshCookieTTL.Seconds()))
}

func (m *CookieManager) Clear(w http.ResponseWriter) {
	m.clearCookie(w, AccessCookieName)
	m.clearCookie(w, RefreshCookieName)
}

func (m *CookieManager) writeCookie(w http.ResponseWriter, name string, value string, expires time.Time, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m != nil && m.secure,
	})
}

func (m *CookieManager) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m != nil && m.secure,
	})
}
