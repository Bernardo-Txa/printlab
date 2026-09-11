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

func TestInfinitePayCreateCheckoutRejectsUnsafeOrInvalidResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{name: "missing url", status: http.StatusOK, body: `{}`, want: ErrInvalidCheckoutURL},
		{name: "http url", status: http.StatusOK, body: `{"url":"http://checkout.infinitepay.com.br/test"}`, want: ErrInvalidCheckoutURL},
		{name: "wrong host", status: http.StatusOK, body: `{"url":"https://evil.example/test"}`, want: ErrInvalidCheckoutURL},
		{name: "invalid json", status: http.StatusOK, body: `{`, want: ErrProviderUnavailable},
		{name: "bad request", status: http.StatusBadRequest, body: `{}`, want: ErrProviderUnavailable},
		{name: "server error", status: http.StatusInternalServerError, body: `{}`, want: ErrProviderUnavailable},
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
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
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
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
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
		name   string
		status int
		body   string
	}{
		{name: "missing amount when paid", status: http.StatusOK, body: `{"success":true,"paid":true,"paid_amount":1510}`},
		{name: "invalid json", status: http.StatusOK, body: `{`},
		{name: "bad request", status: http.StatusBadRequest, body: `{}`},
		{name: "server error", status: http.StatusInternalServerError, body: `{}`},
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
			if !errors.Is(err, ErrProviderUnavailable) {
				t.Fatalf("expected ErrProviderUnavailable, got %v", err)
			}
		})
	}
}

func assertJSONValue(t *testing.T, payload map[string]any, key string, want string) {
	t.Helper()
	if got, ok := payload[key].(string); !ok || got != want {
		t.Fatalf("expected JSON field %s=%q, got %#v", key, want, payload[key])
	}
}
