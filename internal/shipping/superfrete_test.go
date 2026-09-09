package shipping

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

func TestSuperFreteClientSendsProductsPayloadAndHeaders(t *testing.T) {
	secretToken := "superfrete-test-token"
	var gotPath string
	var gotAuthorization string
	var gotUserAgent string
	var gotContentType string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("expected request JSON, got %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(superFreteResponseJSON("1", "PAC", "18.90", "12", "16", "24", "0.47", 5)))
	}))
	defer server.Close()

	client := newTestSuperFreteClient(t, server.URL, secretToken)
	quotes, err := client.Calculate(context.Background(), SuperFreteCalculatorRequest{
		FromPostalCode: "01153000",
		ToPostalCode:   "20020050",
		Services:       "1,2",
		Products: []SuperFreteProduct{
			{Quantity: 2, WeightKG: 0.285, HeightCM: 9, WidthCM: 10.5, LengthCM: 21},
		},
	})
	if err != nil {
		t.Fatalf("expected quote response, got %v", err)
	}

	if gotPath != superFreteCalculatorPath {
		t.Fatalf("expected calculator path, got %q", gotPath)
	}
	if gotAuthorization != "Bearer "+secretToken {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "PrintLab/1.0 (integracao@example.com)" {
		t.Fatalf("expected User-Agent, got %q", gotUserAgent)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", gotContentType)
	}
	if _, ok := gotPayload["products"]; !ok {
		t.Fatalf("expected products payload, got %#v", gotPayload)
	}
	if _, ok := gotPayload["package"]; ok {
		t.Fatalf("expected package not to be sent with products payload, got %#v", gotPayload)
	}
	options := gotPayload["options"].(map[string]any)
	if options["own_hand"] != false || options["receipt"] != false || options["use_insurance_value"] != false {
		t.Fatalf("expected disabled additional services, got %#v", options)
	}
	if len(quotes) != 1 || quotes[0].PriceCents != 1890 || quotes[0].Package == nil || quotes[0].Package.HeightMM != 120 {
		t.Fatalf("expected parsed SuperFrete quote, got %#v", quotes)
	}
}

func TestSuperFreteClientSendsPackagePayload(t *testing.T) {
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("expected request JSON, got %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(superFreteResponseJSON("2", "SEDEX", "31.40", "13", "17", "25", "0.72", 2)))
	}))
	defer server.Close()

	client := newTestSuperFreteClient(t, server.URL, "superfrete-test-token")
	quotes, err := client.Calculate(context.Background(), SuperFreteCalculatorRequest{
		FromPostalCode: "01153000",
		ToPostalCode:   "20020050",
		Services:       "2",
		Package: &SuperFretePackage{
			WeightKG: 0.72,
			HeightCM: 13,
			WidthCM:  17,
			LengthCM: 25,
		},
	})
	if err != nil {
		t.Fatalf("expected quote response, got %v", err)
	}

	if _, ok := gotPayload["package"]; !ok {
		t.Fatalf("expected package payload, got %#v", gotPayload)
	}
	if _, ok := gotPayload["products"]; ok {
		t.Fatalf("expected products not to be sent with package payload, got %#v", gotPayload)
	}
	if len(quotes) != 1 || quotes[0].ServiceCode != "2" || quotes[0].PriceCents != 3140 {
		t.Fatalf("expected parsed package quote, got %#v", quotes)
	}
}

func TestSuperFreteClientSkipsServiceErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"name":"PAC","price":"18.90","delivery_time":5,"company":{"name":"Correios"},"packages":[{"dimensions":{"height":"12","width":"16","length":"24"},"weight":"0.47"}],"has_error":false},
			{"id":2,"name":"SEDEX","price":"31.40","delivery_time":2,"company":{"name":"Correios"},"packages":[],"has_error":true}
		]`))
	}))
	defer server.Close()

	client := newTestSuperFreteClient(t, server.URL, "superfrete-test-token")
	quotes, err := client.Calculate(context.Background(), SuperFreteCalculatorRequest{
		FromPostalCode: "01153000",
		ToPostalCode:   "20020050",
		Services:       "1,2",
		Package:        &SuperFretePackage{WeightKG: 0.47, HeightCM: 12, WidthCM: 16, LengthCM: 24},
	})
	if err != nil {
		t.Fatalf("expected quote response, got %v", err)
	}
	if len(quotes) != 1 || quotes[0].ServiceCode != "1" {
		t.Fatalf("expected only valid service, got %#v", quotes)
	}
}

func TestSuperFreteClientHandlesHTTPAndJSONErrorsSafely(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "bad request", status: http.StatusBadRequest, body: `{"error":"invalid"}`},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"error":"unauthorized"}`},
		{name: "server error", status: http.StatusServiceUnavailable, body: `{"error":"unavailable"}`},
		{name: "invalid json", status: http.StatusOK, body: `{`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretToken := "superfrete-test-token"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestSuperFreteClient(t, server.URL, secretToken)
			_, err := client.Calculate(context.Background(), SuperFreteCalculatorRequest{
				FromPostalCode: "01153000",
				ToPostalCode:   "20020050",
				Services:       "1",
				Package:        &SuperFretePackage{WeightKG: 0.47, HeightCM: 12, WidthCM: 16, LengthCM: 24},
			})
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("expected ErrUnavailable, got %v", err)
			}
			if strings.Contains(err.Error(), secretToken) {
				t.Fatal("expected error not to expose token")
			}
		})
	}
}

func TestSuperFreteClientTimeoutDoesNotExposeToken(t *testing.T) {
	secretToken := "superfrete-test-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestSuperFreteClient(t, server.URL, secretToken)
	client.httpClient.Timeout = 1 * time.Millisecond

	_, err := client.Calculate(context.Background(), SuperFreteCalculatorRequest{
		FromPostalCode: "01153000",
		ToPostalCode:   "20020050",
		Services:       "1",
		Package:        &SuperFretePackage{WeightKG: 0.47, HeightCM: 12, WidthCM: 16, LengthCM: 24},
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Fatal("expected timeout error not to expose token")
	}
}

func newTestSuperFreteClient(t *testing.T, baseURL string, token string) *SuperFreteClient {
	t.Helper()

	client, err := NewSuperFreteClient(SuperFreteClientConfig{
		Environment:  "sandbox",
		APIToken:     token,
		ContactEmail: "integracao@example.com",
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("expected test SuperFrete client, got %v", err)
	}

	return client
}

func superFreteResponseJSON(id string, name string, price string, height string, width string, length string, weight string, deliveryTime int) string {
	return `[
		{
			"id":` + id + `,
			"name":"` + name + `",
			"price":"` + price + `",
			"currency":"R$",
			"delivery_time":` + stringInt(deliveryTime) + `,
			"packages":[
				{
					"price":"` + price + `",
					"format":"box",
					"dimensions":{
						"height":"` + height + `",
						"width":"` + width + `",
						"length":"` + length + `"
					},
					"weight":"` + weight + `"
				}
			],
			"company":{"name":"Correios"},
			"has_error":false
		}
	]`
}

func stringInt(value int) string {
	if value == 0 {
		return "0"
	}

	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}

	return string(digits)
}
