package payments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInfinitePayCreateCheckoutSendsExpectedPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/links" {
			t.Fatalf("expected POST /links, got %s %s", r.Method, r.URL.Path)
		}
		if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
			t.Fatalf("expected JSON content type, got %q", contentType)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("expected JSON payload, got %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://checkout.infinitepay.com.br/checkout-slug"}`))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Second
	client := NewInfinitePayClient(
		WithInfinitePayBaseURL(server.URL),
		WithInfinitePayHTTPClient(httpClient),
	)

	result, err := client.CreateCheckout(context.Background(), CheckoutRequest{
		Handle:      "printlab",
		RedirectURL: "https://printlab.example/pagamento/retorno",
		OrderNSU:    "22222222-2222-2222-2222-222222222222",
		Items: []CheckoutItem{
			{Quantity: 3, PriceCents: 1990, Description: "Produto Real - Padrao"},
			{Quantity: 1, PriceCents: 1000, Description: "Frete - SEDEX"},
		},
		Customer: CheckoutCustomer{
			Name:  "Joao Silva",
			Email: "joao@example.com",
			Phone: "+5527999999999",
		},
		Address: CheckoutAddress{
			PostalCode:   "29100000",
			Street:       "Rua Um",
			Neighborhood: "Centro",
			Number:       "12A",
			Complement:   "Apto 302",
		},
	})
	if err != nil {
		t.Fatalf("expected checkout URL, got %v", err)
	}
	if result.URL != "https://checkout.infinitepay.com.br/checkout-slug" {
		t.Fatalf("expected checkout URL, got %q", result.URL)
	}

	assertJSONValue(t, payload, "handle", "printlab")
	assertJSONValue(t, payload, "redirect_url", "https://printlab.example/pagamento/retorno")
	assertJSONValue(t, payload, "order_nsu", "22222222-2222-2222-2222-222222222222")
	if _, ok := payload["webhook_url"]; ok {
		t.Fatal("expected checkout payload not to send webhook_url")
	}
	if _, ok := payload["cpf"]; ok {
		t.Fatal("expected checkout payload not to send cpf")
	}

	items := payload["items"].([]any)
	firstItem := items[0].(map[string]any)
	if firstItem["quantity"] != float64(3) || firstItem["price"] != float64(1990) || firstItem["description"] != "Produto Real - Padrao" {
		t.Fatalf("expected unit-price item payload, got %#v", firstItem)
	}
	secondItem := items[1].(map[string]any)
	if secondItem["quantity"] != float64(1) || secondItem["price"] != float64(1000) || secondItem["description"] != "Frete - SEDEX" {
		t.Fatalf("expected shipping item payload, got %#v", secondItem)
	}

	customer := payload["customer"].(map[string]any)
	assertJSONValue(t, customer, "name", "Joao Silva")
	assertJSONValue(t, customer, "email", "joao@example.com")
	assertJSONValue(t, customer, "phone_number", "+5527999999999")
	if _, ok := customer["cpf"]; ok {
		t.Fatal("expected customer payload not to include cpf")
	}

	address := payload["address"].(map[string]any)
	assertJSONValue(t, address, "cep", "29100000")
	assertJSONValue(t, address, "street", "Rua Um")
	assertJSONValue(t, address, "neighborhood", "Centro")
	assertJSONValue(t, address, "number", "12A")
	assertJSONValue(t, address, "complement", "Apto 302")
	if _, ok := address["city"]; ok {
		t.Fatal("expected address payload not to include city")
	}
	if _, ok := address["state"]; ok {
		t.Fatal("expected address payload not to include state")
	}
}

func TestInfinitePayCreateCheckoutAcceptsCurrentCheckoutHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://checkout.infinitepay.io/checkout-slug"}`))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Second
	client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))

	result, err := client.CreateCheckout(context.Background(), CheckoutRequest{})
	if err != nil {
		t.Fatalf("expected current InfinitePay checkout host to be valid, got %v", err)
	}
	if result.URL != "https://checkout.infinitepay.io/checkout-slug" {
		t.Fatalf("expected current checkout URL, got %q", result.URL)
	}
}

func TestInfinitePayCreateCheckoutRejectsUnsafeOrInvalidResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		want     error
		category string
	}{
		{name: "missing url", status: http.StatusOK, body: `{}`, want: ErrInvalidCheckoutURL, category: ProviderCategoryInvalidCheckoutURL},
		{name: "http url", status: http.StatusOK, body: `{"url":"http://checkout.infinitepay.com.br/test?ref=private"}`, want: ErrInvalidCheckoutURL, category: ProviderCategoryInvalidCheckoutURL},
		{name: "wrong host", status: http.StatusOK, body: `{"url":"https://evil.example/test?ref=private"}`, want: ErrInvalidCheckoutURL, category: ProviderCategoryInvalidCheckoutURL},
		{name: "invalid json", status: http.StatusOK, body: `{`, want: ErrProviderUnavailable, category: ProviderCategoryInvalidJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			httpClient := server.Client()
			httpClient.Timeout = time.Second
			client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))
			_, err := client.CreateCheckout(context.Background(), CheckoutRequest{})
			assertProviderError(t, err, tt.want, ProviderOperationCreateCheckout, tt.category, 0)
			assertSafeProviderDiagnosticString(t, err.Error())
			assertSafeProviderDiagnosticString(t, ProviderLogFields(err))
		})
	}
}

func TestInfinitePayClassifiesHTTPStatusDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		category string
	}{
		{name: "400", status: http.StatusBadRequest, category: ProviderCategoryHTTP400},
		{name: "401", status: http.StatusUnauthorized, category: ProviderCategoryHTTP401},
		{name: "403", status: http.StatusForbidden, category: ProviderCategoryHTTP403},
		{name: "404", status: http.StatusNotFound, category: ProviderCategoryHTTP404},
		{name: "409", status: http.StatusConflict, category: ProviderCategoryHTTP409},
		{name: "422", status: http.StatusUnprocessableEntity, category: ProviderCategoryHTTP422},
		{name: "429", status: http.StatusTooManyRequests, category: ProviderCategoryHTTP429},
		{name: "500", status: http.StatusInternalServerError, category: ProviderCategoryHTTP5xx},
		{name: "503", status: http.StatusServiceUnavailable, category: ProviderCategoryHTTP5xx},
	}

	for _, tt := range tests {
		t.Run("create_checkout_"+tt.name, func(t *testing.T) {
			client, closeServer := newInfinitePayStatusClient(t, tt.status)
			defer closeServer()

			_, err := client.CreateCheckout(context.Background(), CheckoutRequest{})
			assertProviderError(t, err, ErrProviderUnavailable, ProviderOperationCreateCheckout, tt.category, tt.status)
			assertSafeProviderDiagnosticString(t, err.Error())
			assertSafeProviderDiagnosticString(t, ProviderLogFields(err))
		})

		t.Run("payment_check_"+tt.name, func(t *testing.T) {
			client, closeServer := newInfinitePayStatusClient(t, tt.status)
			defer closeServer()

			_, err := client.CheckPayment(context.Background(), PaymentCheckRequest{})
			assertProviderError(t, err, ErrProviderUnavailable, ProviderOperationPaymentCheck, tt.category, tt.status)
			assertSafeProviderDiagnosticString(t, err.Error())
			assertSafeProviderDiagnosticString(t, ProviderLogFields(err))
		})
	}
}

func TestInfinitePayCreateCheckoutTimeoutReturnsProviderUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"url":"https://checkout.infinitepay.com.br/checkout-slug"}`))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Millisecond
	client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))

	_, err := client.CreateCheckout(context.Background(), CheckoutRequest{})
	assertProviderError(t, err, ErrProviderUnavailable, ProviderOperationCreateCheckout, ProviderCategoryTimeout, 0)
	assertSafeProviderDiagnosticString(t, err.Error())
}

func TestInfinitePayCreateCheckoutNetworkErrorReturnsProviderDiagnostics(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		}),
	}
	client := NewInfinitePayClient(WithInfinitePayBaseURL("https://api.checkout.infinitepay.io"), WithInfinitePayHTTPClient(httpClient))

	_, err := client.CreateCheckout(context.Background(), CheckoutRequest{})
	assertProviderError(t, err, ErrProviderUnavailable, ProviderOperationCreateCheckout, ProviderCategoryNetworkError, 0)
	assertSafeProviderDiagnosticString(t, err.Error())
}

func TestInfinitePayCheckPaymentSendsExpectedPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/payment_check" {
			t.Fatalf("expected POST /payment_check, got %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("expected JSON payload, got %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"paid":true,"amount":1500,"paid_amount":1510,"installments":1,"capture_method":"pix"}`))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Second
	client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))

	result, err := client.CheckPayment(context.Background(), PaymentCheckRequest{
		Handle:         "printlab",
		OrderNSU:       "22222222-2222-2222-2222-222222222222",
		TransactionNSU: "txn_123",
		Slug:           "slug_123",
	})
	if err != nil {
		t.Fatalf("expected payment check result, got %v", err)
	}
	if !result.Success || !result.Paid || result.AmountCents != 1500 || result.PaidAmountCents != 1510 || result.Installments == nil || *result.Installments != 1 || result.CaptureMethod != "pix" {
		t.Fatalf("unexpected payment check result: %#v", result)
	}
	assertJSONValue(t, payload, "handle", "printlab")
	assertJSONValue(t, payload, "order_nsu", "22222222-2222-2222-2222-222222222222")
	assertJSONValue(t, payload, "transaction_nsu", "txn_123")
	assertJSONValue(t, payload, "slug", "slug_123")
}

func TestInfinitePayCheckPaymentHandlesProviderFailures(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		category string
	}{
		{name: "missing amount when paid", status: http.StatusOK, body: `{"success":true,"paid":true,"paid_amount":1510}`, category: ProviderCategoryInvalidJSON},
		{name: "invalid json", status: http.StatusOK, body: `{`, category: ProviderCategoryInvalidJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			httpClient := server.Client()
			httpClient.Timeout = time.Second
			client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))
			_, err := client.CheckPayment(context.Background(), PaymentCheckRequest{})
			assertProviderError(t, err, ErrProviderUnavailable, ProviderOperationPaymentCheck, tt.category, 0)
			assertSafeProviderDiagnosticString(t, err.Error())
		})
	}
}

func newInfinitePayStatusClient(t *testing.T, status int) (*InfinitePayClient, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"invalid customer Joao Silva joao@example.com +5527999999999 Rua Um https://checkout.infinitepay.com.br/checkout-slug","error":"payload contains private address","code":"invalid_handle"}`))
	}))

	httpClient := server.Client()
	httpClient.Timeout = time.Second
	client := NewInfinitePayClient(WithInfinitePayBaseURL(server.URL), WithInfinitePayHTTPClient(httpClient))
	return client, server.Close
}

func assertProviderError(t *testing.T, err error, cause error, operation string, category string, status int) {
	t.Helper()
	if !errors.Is(err, cause) {
		t.Fatalf("expected %v, got %v", cause, err)
	}

	details, ok := ProviderErrorDetailsFor(err)
	if !ok {
		t.Fatalf("expected provider details, got %T: %v", err, err)
	}
	if details.Provider != ProviderInfinitePay {
		t.Fatalf("expected provider %q, got %q", ProviderInfinitePay, details.Provider)
	}
	if details.Operation != operation {
		t.Fatalf("expected operation %q, got %q", operation, details.Operation)
	}
	if details.Category != category {
		t.Fatalf("expected category %q, got %q", category, details.Category)
	}
	if details.StatusCode != status {
		t.Fatalf("expected status %d, got %d", status, details.StatusCode)
	}
}

func assertSafeProviderDiagnosticString(t *testing.T, value string) {
	t.Helper()
	for _, leaked := range []string{
		"Joao",
		"joao@example.com",
		"+5527999999999",
		"Rua Um",
		"29100000",
		"12A",
		"Apto 302",
		"https://checkout.infinitepay.com.br/checkout-slug",
		"http://checkout.infinitepay.com.br/test",
		"https://evil.example/test",
		"ref=private",
		"txn_123",
	} {
		if strings.Contains(value, leaked) {
			t.Fatalf("expected safe diagnostic not to contain %q, got %q", leaked, value)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func assertJSONValue(t *testing.T, payload map[string]any, key string, want string) {
	t.Helper()
	if got, ok := payload[key].(string); !ok || got != want {
		t.Fatalf("expected JSON field %s=%q, got %#v", key, want, payload[key])
	}
}
