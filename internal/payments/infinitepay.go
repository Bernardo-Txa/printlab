package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	infinitePayBaseURL       = "https://api.checkout.infinitepay.io"
	infinitePayClientTimeout = 8 * time.Second
	maxProviderResponseBytes = 1 << 20
)

type Gateway interface {
	CreateCheckout(ctx context.Context, request CheckoutRequest) (CheckoutCreated, error)
	CheckPayment(ctx context.Context, request PaymentCheckRequest) (PaymentCheckResult, error)
}

type InfinitePayClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type InfinitePayClientOption func(*InfinitePayClient)

func NewInfinitePayClient(options ...InfinitePayClientOption) *InfinitePayClient {
	baseURL, _ := url.Parse(infinitePayBaseURL)
	client := &InfinitePayClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: infinitePayClientTimeout,
		},
	}
	for _, option := range options {
		option(client)
	}
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: infinitePayClientTimeout}
	}
	if client.baseURL == nil {
		client.baseURL, _ = url.Parse(infinitePayBaseURL)
	}

	return client
}

func WithInfinitePayBaseURL(rawURL string) InfinitePayClientOption {
	return func(client *InfinitePayClient) {
		parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(rawURL), "/"))
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			client.baseURL = parsed
		}
	}
}

func WithInfinitePayHTTPClient(httpClient *http.Client) InfinitePayClientOption {
	return func(client *InfinitePayClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func (c *InfinitePayClient) CreateCheckout(ctx context.Context, request CheckoutRequest) (CheckoutCreated, error) {
	body := infinitePayCheckoutRequest{
		Handle:      request.Handle,
		RedirectURL: request.RedirectURL,
		OrderNSU:    request.OrderNSU,
		Customer: infinitePayCustomer{
			Name:        request.Customer.Name,
			Email:       request.Customer.Email,
			PhoneNumber: request.Customer.Phone,
		},
		Address: infinitePayAddress{
			CEP:          request.Address.PostalCode,
			Street:       request.Address.Street,
			Neighborhood: request.Address.Neighborhood,
			Number:       request.Address.Number,
			Complement:   request.Address.Complement,
		},
	}
	for _, item := range request.Items {
		body.Items = append(body.Items, infinitePayCheckoutItem{
			Quantity:    item.Quantity,
			Price:       item.PriceCents,
			Description: item.Description,
		})
	}

	var response infinitePayCheckoutResponse
	if err := c.postJSON(ctx, ProviderOperationCreateCheckout, "/links", body, &response); err != nil {
		return CheckoutCreated{}, err
	}
	if err := ValidateCheckoutURL(response.URL); err != nil {
		return CheckoutCreated{}, providerInvalidCheckoutURLError(response.URL)
	}

	return CheckoutCreated{URL: response.URL}, nil
}

func (c *InfinitePayClient) CheckPayment(ctx context.Context, request PaymentCheckRequest) (PaymentCheckResult, error) {
	body := infinitePayPaymentCheckRequest{
		Handle:         request.Handle,
		OrderNSU:       request.OrderNSU,
		TransactionNSU: request.TransactionNSU,
		Slug:           request.Slug,
	}

	var response infinitePayPaymentCheckResponse
	if err := c.postJSON(ctx, ProviderOperationPaymentCheck, "/payment_check", body, &response); err != nil {
		return PaymentCheckResult{}, err
	}
	if response.Paid {
		if response.Amount == nil || response.PaidAmount == nil {
			return PaymentCheckResult{}, providerInvalidJSONError(ProviderOperationPaymentCheck)
		}
		if response.Installments != nil && *response.Installments <= 0 {
			return PaymentCheckResult{}, providerInvalidJSONError(ProviderOperationPaymentCheck)
		}
	}

	result := PaymentCheckResult{
		Success:       response.Success,
		Paid:          response.Paid,
		Installments:  response.Installments,
		CaptureMethod: strings.TrimSpace(response.CaptureMethod),
	}
	if response.Amount != nil {
		result.AmountCents = *response.Amount
	}
	if response.PaidAmount != nil {
		result.PaidAmountCents = *response.PaidAmount
	}

	return result, nil
}

func (c *InfinitePayClient) postJSON(ctx context.Context, operation string, endpoint string, body any, response any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return NewProviderError(operation, ProviderCategoryUnknown, 0, ErrProviderUnavailable)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(endpoint), bytes.NewReader(payload))
	if err != nil {
		return NewProviderError(operation, ProviderCategoryUnknown, 0, ErrProviderUnavailable)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return providerNetworkError(operation, err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return providerHTTPError(operation, httpResponse.StatusCode, httpResponse.Body)
	}

	decoder := json.NewDecoder(io.LimitReader(httpResponse.Body, maxProviderResponseBytes))
	if err := decoder.Decode(response); err != nil {
		return providerInvalidJSONError(operation)
	}

	return nil
}

func (c *InfinitePayClient) endpoint(path string) string {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return endpoint.String()
}

type infinitePayCheckoutRequest struct {
	Handle      string                    `json:"handle"`
	RedirectURL string                    `json:"redirect_url"`
	OrderNSU    string                    `json:"order_nsu"`
	Items       []infinitePayCheckoutItem `json:"items"`
	Customer    infinitePayCustomer       `json:"customer"`
	Address     infinitePayAddress        `json:"address"`
}

type infinitePayCheckoutItem struct {
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`
	Description string `json:"description"`
}

type infinitePayCustomer struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}

type infinitePayAddress struct {
	CEP          string `json:"cep"`
	Street       string `json:"street"`
	Neighborhood string `json:"neighborhood"`
	Number       string `json:"number"`
	Complement   string `json:"complement"`
}

type infinitePayCheckoutResponse struct {
	URL string `json:"url"`
}

type infinitePayPaymentCheckRequest struct {
	Handle         string `json:"handle"`
	OrderNSU       string `json:"order_nsu"`
	TransactionNSU string `json:"transaction_nsu"`
	Slug           string `json:"slug"`
}

type infinitePayPaymentCheckResponse struct {
	Success       bool   `json:"success"`
	Paid          bool   `json:"paid"`
	Amount        *int64 `json:"amount"`
	PaidAmount    *int64 `json:"paid_amount"`
	Installments  *int   `json:"installments"`
	CaptureMethod string `json:"capture_method"`
}
