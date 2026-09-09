package customers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	ViaCEPBaseURL        = "https://viacep.com.br"
	defaultViaCEPTimeout = 3 * time.Second
)

var (
	ErrPostalCodeNotFound    = errors.New("postal code not found")
	ErrPostalCodeUnavailable = errors.New("postal code lookup unavailable")
)

type PostalCodeAddress struct {
	Street   string
	District string
	City     string
	State    string
}

type PostalCodeLookupError struct {
	Reason     string
	StatusCode int
}

func (e *PostalCodeLookupError) Error() string {
	if e == nil {
		return ErrPostalCodeUnavailable.Error()
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: %s status=%d", ErrPostalCodeUnavailable, e.Reason, e.StatusCode)
	}

	return fmt.Sprintf("%s: %s", ErrPostalCodeUnavailable, e.Reason)
}

func (e *PostalCodeLookupError) Unwrap() error {
	return ErrPostalCodeUnavailable
}

type ViaCEPClient struct {
	httpClient *http.Client
	baseURL    string
}

type ViaCEPOption func(*ViaCEPClient)

func WithViaCEPHTTPClient(httpClient *http.Client) ViaCEPOption {
	return func(client *ViaCEPClient) {
		if httpClient == nil {
			return
		}

		cloned := *httpClient
		if cloned.Timeout == 0 {
			cloned.Timeout = defaultViaCEPTimeout
		}
		client.httpClient = &cloned
	}
}

func WithViaCEPBaseURL(baseURL string) ViaCEPOption {
	return func(client *ViaCEPClient) {
		if strings.TrimSpace(baseURL) != "" {
			client.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func NewViaCEPClient(options ...ViaCEPOption) *ViaCEPClient {
	client := &ViaCEPClient{
		httpClient: &http.Client{Timeout: defaultViaCEPTimeout},
		baseURL:    ViaCEPBaseURL,
	}
	for _, option := range options {
		option(client)
	}

	return client
}

func (c *ViaCEPClient) Lookup(ctx context.Context, postalCode string) (PostalCodeAddress, error) {
	if c == nil || c.httpClient == nil || c.baseURL == "" {
		return PostalCodeAddress{}, postalCodeUnavailable("not_configured", 0)
	}

	normalized, err := NormalizePostalCode(postalCode)
	if err != nil {
		return PostalCodeAddress{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ws/"+normalized+"/json/", nil)
	if err != nil {
		return PostalCodeAddress{}, postalCodeUnavailable("request_build_failed", 0)
	}
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		if isTimeoutError(err) {
			return PostalCodeAddress{}, postalCodeUnavailable("timeout", 0)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return PostalCodeAddress{}, ctx.Err()
		}

		return PostalCodeAddress{}, postalCodeUnavailable("request_failed", 0)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusBadRequest {
		return PostalCodeAddress{}, ErrInvalidDetails
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return PostalCodeAddress{}, postalCodeUnavailable("status", response.StatusCode)
	}

	var external viaCEPResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&external); err != nil {
		return PostalCodeAddress{}, postalCodeUnavailable("invalid_json", 0)
	}
	if external.Error {
		return PostalCodeAddress{}, ErrPostalCodeNotFound
	}

	state, err := NormalizeState(external.State)
	if err != nil || strings.TrimSpace(external.City) == "" {
		return PostalCodeAddress{}, postalCodeUnavailable("invalid_response", 0)
	}

	return PostalCodeAddress{
		Street:   strings.TrimSpace(external.Street),
		District: strings.TrimSpace(external.District),
		City:     strings.TrimSpace(external.City),
		State:    state,
	}, nil
}

func postalCodeUnavailable(reason string, statusCode int) error {
	return &PostalCodeLookupError{Reason: reason, StatusCode: statusCode}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

type viaCEPResponse struct {
	Street   string `json:"logradouro"`
	District string `json:"bairro"`
	City     string `json:"localidade"`
	State    string `json:"uf"`
	Error    bool   `json:"erro"`
}
