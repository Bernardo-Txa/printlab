package customers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestViaCEPClientLookupValidPostalCode(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cep":"01001-000",
			"logradouro":"Praca da Se",
			"bairro":"Se",
			"localidade":"Sao Paulo",
			"uf":"SP",
			"ibge":"3550308",
			"ddd":"11",
			"siafi":"7107"
		}`))
	}))
	defer server.Close()

	client := NewViaCEPClient(WithViaCEPBaseURL(server.URL))
	address, err := client.Lookup(context.Background(), "01001-000")
	if err != nil {
		t.Fatalf("expected valid postal code lookup, got %v", err)
	}

	if gotPath != "/ws/01001000/json/" {
		t.Fatalf("expected normalized ViaCEP path, got %q", gotPath)
	}
	if address.Street != "Praca da Se" || address.District != "Se" || address.City != "Sao Paulo" || address.State != "SP" {
		t.Fatalf("expected mapped address only, got %#v", address)
	}
}

func TestViaCEPClientRejectsInvalidPostalCodeWithoutCallingServer(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	defer server.Close()

	client := NewViaCEPClient(WithViaCEPBaseURL(server.URL))
	_, err := client.Lookup(context.Background(), "123")
	if !errors.Is(err, ErrInvalidDetails) {
		t.Fatalf("expected ErrInvalidDetails, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected invalid CEP not to call ViaCEP, got %d calls", calls)
	}
}

func TestViaCEPClientHandlesMissingPostalCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"erro":true}`))
	}))
	defer server.Close()

	client := NewViaCEPClient(WithViaCEPBaseURL(server.URL))
	_, err := client.Lookup(context.Background(), "99999999")
	if !errors.Is(err, ErrPostalCodeNotFound) {
		t.Fatalf("expected ErrPostalCodeNotFound, got %v", err)
	}
}

func TestViaCEPClientHandlesHTTPStatusSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewViaCEPClient(WithViaCEPBaseURL(server.URL))
	_, err := client.Lookup(context.Background(), "01001000")
	if !errors.Is(err, ErrPostalCodeUnavailable) {
		t.Fatalf("expected ErrPostalCodeUnavailable, got %v", err)
	}

	var lookupErr *PostalCodeLookupError
	if !errors.As(err, &lookupErr) || lookupErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected safe status classification, got %v", err)
	}
	if strings.Contains(err.Error(), "01001000") {
		t.Fatal("expected error not to expose CEP")
	}
}

func TestViaCEPClientHandlesTimeoutSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"localidade":"Sao Paulo","uf":"SP"}`))
	}))
	defer server.Close()

	client := NewViaCEPClient(
		WithViaCEPBaseURL(server.URL),
		WithViaCEPHTTPClient(&http.Client{Timeout: time.Millisecond}),
	)
	_, err := client.Lookup(context.Background(), "01001000")
	if !errors.Is(err, ErrPostalCodeUnavailable) {
		t.Fatalf("expected ErrPostalCodeUnavailable, got %v", err)
	}

	var lookupErr *PostalCodeLookupError
	if !errors.As(err, &lookupErr) || lookupErr.Reason != "timeout" {
		t.Fatalf("expected timeout classification, got %v", err)
	}
}

func TestViaCEPClientHandlesInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()

	client := NewViaCEPClient(WithViaCEPBaseURL(server.URL))
	_, err := client.Lookup(context.Background(), "01001000")
	if !errors.Is(err, ErrPostalCodeUnavailable) {
		t.Fatalf("expected ErrPostalCodeUnavailable, got %v", err)
	}

	var lookupErr *PostalCodeLookupError
	if !errors.As(err, &lookupErr) || lookupErr.Reason != "invalid_json" {
		t.Fatalf("expected invalid_json classification, got %v", err)
	}
}
