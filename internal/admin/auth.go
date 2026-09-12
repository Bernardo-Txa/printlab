package admin

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

type AuthClient interface {
	SignInWithPassword(ctx context.Context, email string, password string) (AuthUser, error)
}

type SupabaseAuthClientConfig struct {
	SupabaseURL    string
	PublishableKey string
	HTTPClient     *http.Client
}

type SupabaseAuthClient struct {
	baseURL        *url.URL
	publishableKey string
	httpClient     *http.Client
}

func NewSupabaseAuthClient(cfg SupabaseAuthClientConfig) (*SupabaseAuthClient, error) {
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.SupabaseURL), "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" || strings.TrimSpace(cfg.PublishableKey) == "" {
		return nil, ErrAuthUnavailable
	}
	if baseURL.Scheme != "https" && baseURL.Scheme != "http" {
		return nil, ErrAuthUnavailable
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultAuthClientTimeout}
	}

	return &SupabaseAuthClient{
		baseURL:        baseURL,
		publishableKey: strings.TrimSpace(cfg.PublishableKey),
		httpClient:     httpClient,
	}, nil
}

func (c *SupabaseAuthClient) SignInWithPassword(ctx context.Context, email string, password string) (AuthUser, error) {
	if c == nil || c.baseURL == nil || c.httpClient == nil || c.publishableKey == "" {
		return AuthUser{}, ErrAuthUnavailable
	}
	if strings.TrimSpace(email) == "" || password == "" {
		return AuthUser{}, ErrAuthRejected
	}

	payload, err := json.Marshal(supabasePasswordLoginRequest{
		Email:    strings.TrimSpace(email),
		Password: password,
	})
	if err != nil {
		return AuthUser{}, ErrAuthUnavailable
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), bytes.NewReader(payload))
	if err != nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	request.Header.Set("apikey", c.publishableKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return AuthUser{}, ErrAuthRejected
		}
		return AuthUser{}, ErrAuthUnavailable
	}

	var loginResponse supabasePasswordLoginResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxAuthResponseBytes))
	if err := decoder.Decode(&loginResponse); err != nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	if loginResponse.User == nil || !ValidUUID(loginResponse.User.ID) {
		return AuthUser{}, ErrAuthUnavailable
	}

	return AuthUser{ID: normalizeUUID(loginResponse.User.ID)}, nil
}

func (c *SupabaseAuthClient) tokenEndpoint() string {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/auth/v1/token"
	query := endpoint.Query()
	query.Set("grant_type", "password")
	endpoint.RawQuery = query.Encode()
	endpoint.Fragment = ""
	return endpoint.String()
}

func SafeAuthError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrAuthRejected) {
		return ErrAuthRejected
	}
	return ErrAuthUnavailable
}

type supabasePasswordLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type supabasePasswordLoginResponse struct {
	User *struct {
		ID string `json:"id"`
	} `json:"user"`
}
