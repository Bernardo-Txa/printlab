package customerauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAuthClientTimeout = 8 * time.Second
	maxAuthResponseBytes     = 1 << 20
)

type SupabaseClientConfig struct {
	SupabaseURL    string
	PublishableKey string
	HTTPClient     *http.Client
}

type SupabaseClient struct {
	baseURL        *url.URL
	publishableKey string
	httpClient     *http.Client
}

func NewSupabaseClient(cfg SupabaseClientConfig) (*SupabaseClient, error) {
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.SupabaseURL), "/"))
	if err != nil || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" || (baseURL.Scheme != "https" && baseURL.Scheme != "http") || strings.TrimSpace(cfg.PublishableKey) == "" {
		return nil, ErrConfiguration
	}
	client := http.Client{Timeout: defaultAuthClientTimeout}
	if cfg.HTTPClient != nil {
		client = *cfg.HTTPClient
	}
	if client.Timeout <= 0 {
		client.Timeout = defaultAuthClientTimeout
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &SupabaseClient{baseURL: baseURL, publishableKey: strings.TrimSpace(cfg.PublishableKey), httpClient: &client}, nil
}

func (c *SupabaseClient) SignUp(ctx context.Context, input SignUpInput) (SignUpResult, error) {
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return SignUpResult{}, ErrRejected
	}
	payload := map[string]any{
		"email":    strings.TrimSpace(input.Email),
		"password": input.Password,
	}
	if strings.TrimSpace(input.Name) != "" {
		payload["data"] = map[string]string{"full_name": strings.TrimSpace(input.Name)}
	}
	var result signUpResponse
	if err := c.request(ctx, http.MethodPost, "signup", "", input.RedirectURL, payload, &result); err != nil {
		return SignUpResult{}, err
	}
	user := result.User
	if user.ID == "" {
		user = result.AuthUser
	}
	if user.ID == "" {
		return SignUpResult{}, ErrInvalidResponse
	}
	return SignUpResult{User: user}, nil
}

type signUpResponse struct {
	User AuthUser `json:"user"`
	AuthUser
}

func (c *SupabaseClient) SignInWithPassword(ctx context.Context, email string, password string) (AuthSession, error) {
	if strings.TrimSpace(email) == "" || password == "" {
		return AuthSession{}, ErrRejected
	}
	var result AuthSession
	err := c.request(ctx, http.MethodPost, "token", "", "", map[string]string{"email": strings.TrimSpace(email), "password": password}, &result)
	return validSession(result, err)
}

func (c *SupabaseClient) RecoverPassword(ctx context.Context, email string, redirectTo string) error {
	if strings.TrimSpace(email) == "" {
		return ErrRejected
	}
	return c.request(ctx, http.MethodPost, "recover", "", redirectTo, map[string]string{"email": strings.TrimSpace(email)}, &struct{}{})
}

func (c *SupabaseClient) UpdatePassword(ctx context.Context, accessToken string, password string) error {
	if accessToken == "" || password == "" {
		return ErrRejected
	}
	var result AuthUser
	if err := c.request(ctx, http.MethodPut, "user", accessToken, "", map[string]string{"password": password}, &result); err != nil {
		return err
	}
	if result.ID == "" {
		return ErrInvalidResponse
	}
	return nil
}

func (c *SupabaseClient) RefreshSession(ctx context.Context, refreshToken string) (AuthSession, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return AuthSession{}, ErrUnauthenticated
	}
	var result AuthSession
	err := c.request(ctx, http.MethodPost, "token", "", "", map[string]string{"refresh_token": strings.TrimSpace(refreshToken)}, &result)
	return validSession(result, err)
}

func (c *SupabaseClient) GetUser(ctx context.Context, accessToken string) (AuthUser, error) {
	if accessToken == "" {
		return AuthUser{}, ErrUnauthenticated
	}
	var result AuthUser
	if err := c.request(ctx, http.MethodGet, "user", accessToken, "", nil, &result); err != nil {
		if errors.Is(err, ErrRejected) {
			return AuthUser{}, ErrUnauthenticated
		}
		return AuthUser{}, err
	}
	if result.ID == "" || strings.TrimSpace(result.Email) == "" {
		return AuthUser{}, ErrInvalidResponse
	}
	return result, nil
}

func (c *SupabaseClient) Logout(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return ErrUnauthenticated
	}
	return c.request(ctx, http.MethodPost, "logout", accessToken, "", struct{}{}, &struct{}{})
}

func validSession(result AuthSession, err error) (AuthSession, error) {
	if err != nil {
		return AuthSession{}, err
	}
	if result.AccessToken == "" || result.RefreshToken == "" || result.User.ID == "" || strings.TrimSpace(result.User.Email) == "" {
		return AuthSession{}, ErrInvalidResponse
	}
	return result, nil
}

func (c *SupabaseClient) request(ctx context.Context, method string, path string, accessToken string, redirectTo string, payload any, dest any) error {
	if c == nil || c.baseURL == nil || c.httpClient == nil || c.publishableKey == "" {
		return ErrConfiguration
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return ErrConfiguration
		}
		body = bytes.NewReader(data)
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/auth/v1/" + path
	query := endpoint.Query()
	if path == "token" {
		if m, ok := payload.(map[string]string); ok && m["refresh_token"] != "" {
			query.Set("grant_type", "refresh_token")
		} else {
			query.Set("grant_type", "password")
		}
	}
	if strings.TrimSpace(redirectTo) != "" {
		query.Set("redirect_to", strings.TrimSpace(redirectTo))
	}
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return ErrConfiguration
	}
	req.Header.Set("apikey", c.publishableKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusNotFound:
		return ErrRejected
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ErrUnavailable
	}
	if dest == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxAuthResponseBytes))
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAuthResponseBytes+1))
	if err != nil {
		return ErrUnavailable
	}
	if len(data) > maxAuthResponseBytes {
		return ErrInvalidResponse
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	if json.Unmarshal(data, dest) != nil {
		return ErrInvalidResponse
	}
	return nil
}

func SafeProviderError(err error) error {
	if err == nil {
		return nil
	}
	for _, safe := range []error{ErrRejected, ErrRateLimited, ErrInvalidResponse, ErrConfiguration, ErrUnauthenticated} {
		if errors.Is(err, safe) {
			return safe
		}
	}
	return ErrUnavailable
}
