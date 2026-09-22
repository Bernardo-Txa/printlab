package customerauth

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Service struct {
	provider Provider
	cookies  *CookieManager
	siteURL  string
	now      func() time.Time
}

func NewService(provider Provider, cookies *CookieManager, siteURL string) *Service {
	if cookies == nil {
		cookies = NewCookieManager(CookieOptions{})
	}
	return &Service{
		provider: provider,
		cookies:  cookies,
		siteURL:  strings.TrimRight(strings.TrimSpace(siteURL), "/"),
		now:      time.Now,
	}
}

func (s *Service) Available() bool {
	return s != nil && s.provider != nil && s.cookies != nil
}

func (s *Service) SignUp(ctx context.Context, input SignUpInput) error {
	if !s.Available() {
		return ErrUnavailable
	}
	if _, err := s.provider.SignUp(ctx, input); err != nil {
		return SafeProviderError(err)
	}
	return nil
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, email string, password string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	session, err := s.provider.SignInWithPassword(ctx, email, password)
	if err != nil {
		return SafeProviderError(err)
	}
	s.cookies.WriteSession(w, session, s.now().UTC())
	return nil
}

func (s *Service) RecoverPassword(ctx context.Context, email string, redirectTo string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	if err := s.provider.RecoverPassword(ctx, email, redirectTo); err != nil {
		return SafeProviderError(err)
	}
	return nil
}

func (s *Service) CompleteCallbackSession(ctx context.Context, w http.ResponseWriter, session AuthSession) error {
	if !s.Available() {
		return ErrUnavailable
	}
	if strings.TrimSpace(session.AccessToken) == "" || strings.TrimSpace(session.RefreshToken) == "" {
		return ErrInvalidSessionToken
	}
	s.cookies.WriteSession(w, session, s.now().UTC())
	return nil
}

func (s *Service) UpdatePassword(ctx context.Context, w http.ResponseWriter, r *http.Request, password string) error {
	if !s.Available() {
		return ErrUnavailable
	}
	accessToken, ok := s.cookies.ReadAccessToken(r)
	if !ok {
		return ErrUnauthenticated
	}
	if err := s.provider.UpdatePassword(ctx, accessToken, password); err != nil {
		return SafeProviderError(err)
	}
	s.Logout(ctx, w, r)
	return nil
}

func (s *Service) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !s.Available() {
		return
	}
	if accessToken, ok := s.cookies.ReadAccessToken(r); ok {
		_ = s.provider.Logout(ctx, accessToken)
	}
	s.cookies.Clear(w)
}

func (s *Service) ResolveSession(ctx context.Context, w http.ResponseWriter, r *http.Request) (Profile, error) {
	if !s.Available() {
		return Profile{}, ErrUnavailable
	}
	accessToken, hasAccess := s.cookies.ReadAccessToken(r)
	if hasAccess {
		user, err := s.provider.GetUser(ctx, accessToken)
		if err == nil {
			return profileFromUser(user), nil
		}
		if !errors.Is(err, ErrUnauthenticated) && !errors.Is(err, ErrRejected) {
			return Profile{}, SafeProviderError(err)
		}
	}
	refreshToken, hasRefresh := s.cookies.ReadRefreshToken(r)
	if !hasRefresh {
		return Profile{}, ErrUnauthenticated
	}
	session, err := s.provider.RefreshSession(ctx, refreshToken)
	if err != nil {
		s.cookies.Clear(w)
		return Profile{}, SafeProviderError(err)
	}
	s.cookies.WriteSession(w, session, s.now().UTC())
	return profileFromUser(session.User), nil
}

func (s *Service) RedirectURL(r *http.Request, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		path = "/auth/callback"
	}
	base := s.siteURL
	if base == "" && r != nil {
		scheme := "https"
		if r.TLS == nil && strings.HasPrefix(r.Host, "localhost") {
			scheme = "http"
		}
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded == "http" || forwarded == "https" {
			scheme = forwarded
		}
		base = scheme + "://" + r.Host
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return path
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func profileFromUser(user AuthUser) Profile {
	return Profile{ID: user.ID, Name: user.FullName(), Email: strings.TrimSpace(user.Email)}
}
