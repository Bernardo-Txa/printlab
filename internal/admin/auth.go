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
	SignInWithPassword(context.Context, string, string) (AuthSession, error)
	GetUser(context.Context, string) (AuthUser, error)
	EnrollTOTP(context.Context, string) (TOTPEnrollment, error)
	UnenrollFactor(context.Context, string, string) error
	Challenge(context.Context, string, string) (string, error)
	Verify(context.Context, string, string, string, string) (AuthSession, error)
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
	if err != nil || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" || (baseURL.Scheme != "https" && baseURL.Scheme != "http") || strings.TrimSpace(cfg.PublishableKey) == "" {
		return nil, ErrAuthConfiguration
	}
	client := http.Client{Timeout: defaultAuthClientTimeout}
	if cfg.HTTPClient != nil {
		client = *cfg.HTTPClient
	}
	if client.Timeout <= 0 {
		client.Timeout = defaultAuthClientTimeout
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &SupabaseAuthClient{baseURL: baseURL, publishableKey: strings.TrimSpace(cfg.PublishableKey), httpClient: &client}, nil
}

func (c *SupabaseAuthClient) SignInWithPassword(ctx context.Context, email, password string) (AuthSession, error) {
	if strings.TrimSpace(email) == "" || password == "" {
		return AuthSession{}, ErrAuthRejected
	}
	var result AuthSession
	err := c.request(ctx, http.MethodPost, "token", "", map[string]string{"email": strings.TrimSpace(email), "password": password}, &result)
	return validAuthSession(result, err)
}

func (c *SupabaseAuthClient) GetUser(ctx context.Context, token string) (AuthUser, error) {
	var result AuthUser
	if token == "" {
		return result, ErrUnauthenticated
	}
	if err := c.request(ctx, http.MethodGet, "user", token, nil, &result); err != nil {
		if errors.Is(err, ErrAuthRejected) {
			return AuthUser{}, ErrUnauthenticated
		}
		return AuthUser{}, err
	}
	if !ValidUUID(result.ID) {
		return AuthUser{}, ErrAuthInvalidResponse
	}
	result.ID = normalizeUUID(result.ID)
	return result, nil
}

func (c *SupabaseAuthClient) EnrollTOTP(ctx context.Context, token string) (TOTPEnrollment, error) {
	var result TOTPEnrollment
	err := c.request(ctx, http.MethodPost, "factors", token, map[string]string{"factor_type": "totp", "friendly_name": "PrintLab Admin", "issuer": "PrintLab"}, &result)
	if err != nil {
		return TOTPEnrollment{}, err
	}
	if !ValidUUID(result.ID) || result.Type != "totp" || result.TOTP.Secret == "" || !validQRCodeSVG(result.TOTP.QRCode) {
		return TOTPEnrollment{}, ErrAuthInvalidResponse
	}
	return result, nil
}

func validQRCodeSVG(value string) bool {
	qr := strings.TrimSpace(value)
	return qr != "" && strings.Contains(qr, "<svg") && strings.Contains(qr, "</svg>")
}

func (c *SupabaseAuthClient) UnenrollFactor(ctx context.Context, token, factorID string) error {
	if !ValidUUID(factorID) {
		return ErrMFAFactor
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.request(ctx, http.MethodDelete, "factors/"+factorID, token, nil, &result); err != nil {
		return err
	}
	if result.ID != factorID {
		return ErrAuthInvalidResponse
	}
	return nil
}

func (c *SupabaseAuthClient) Challenge(ctx context.Context, token, factorID string) (string, error) {
	if !ValidUUID(factorID) {
		return "", ErrMFAFactor
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.request(ctx, http.MethodPost, "factors/"+factorID+"/challenge", token, struct{}{}, &result); err != nil {
		return "", err
	}
	if !ValidUUID(result.ID) {
		return "", ErrAuthInvalidResponse
	}
	return result.ID, nil
}

func (c *SupabaseAuthClient) Verify(ctx context.Context, token, factorID, challengeID, code string) (AuthSession, error) {
	if !ValidUUID(factorID) || !ValidUUID(challengeID) {
		return AuthSession{}, ErrMFAFactor
	}
	var result AuthSession
	err := c.request(ctx, http.MethodPost, "factors/"+factorID+"/verify", token, map[string]string{"challenge_id": challengeID, "code": code}, &result)
	if errors.Is(err, ErrAuthRejected) {
		return AuthSession{}, ErrMFAInvalidCode
	}
	return validAuthSession(result, err)
}

func validAuthSession(result AuthSession, err error) (AuthSession, error) {
	if err != nil {
		return AuthSession{}, err
	}
	if !ValidUUID(result.User.ID) || result.AccessToken == "" {
		return AuthSession{}, ErrAuthInvalidResponse
	}
	result.User.ID = normalizeUUID(result.User.ID)
	return result, nil
}

func (c *SupabaseAuthClient) request(ctx context.Context, method, path, token string, payload, dest any) error {
	if c == nil || c.baseURL == nil || c.httpClient == nil || c.publishableKey == "" {
		return ErrAuthConfiguration
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ErrAuthConfiguration
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/auth/v1/" + path
	if path == "token" {
		endpoint.RawQuery = "grant_type=password"
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return ErrAuthConfiguration
	}
	req.Header.Set("apikey", c.publishableKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return ErrAuthUnavailable
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusTooManyRequests:
		return ErrAuthRateLimited
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity:
		return ErrAuthRejected
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ErrAuthUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAuthResponseBytes+1))
	if err != nil {
		return ErrAuthUnavailable
	}
	if len(data) > maxAuthResponseBytes || json.Unmarshal(data, dest) != nil {
		return ErrAuthInvalidResponse
	}
	return nil
}

func SafeAuthError(err error) error {
	if err == nil {
		return nil
	}
	for _, safe := range []error{ErrAuthRejected, ErrAuthRateLimited, ErrAuthInvalidResponse, ErrAuthConfiguration} {
		if errors.Is(err, safe) {
			return safe
		}
	}
	return ErrAuthUnavailable
}
