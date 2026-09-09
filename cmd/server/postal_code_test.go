package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/database"
)

func TestPostalCodeLookupReturnsSafeAddressJSON(t *testing.T) {
	lookup := &fakePostalCodeLookup{
		address: customers.PostalCodeAddress{
			Street:   "Praca da Se",
			District: "Se",
			City:     "Sao Paulo",
			State:    "SP",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/cep/01001-000", nil)
	rec := httptest.NewRecorder()

	newTestHandlerWithPostalCode(t, lookup).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if lookup.lastPostalCode != "01001-000" {
		t.Fatalf("expected handler to pass route CEP, got %q", lookup.lastPostalCode)
	}
	if got := rec.Header().Get("Cache-Control"); got != checkoutPrivateCacheControl {
		t.Fatalf("expected private no-store cache control, got %q", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("expected JSON response, got %q", got)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body, got %v", err)
	}
	if len(body) != 4 || body["street"] != "Praca da Se" || body["district"] != "Se" || body["city"] != "Sao Paulo" || body["state"] != "SP" {
		t.Fatalf("expected address fields only, got %#v", body)
	}
	for _, forbidden := range []string{"ibge", "ddd", "siafi", "gia", "regiao"} {
		if _, ok := body[forbidden]; ok {
			t.Fatalf("expected response not to expose %q", forbidden)
		}
	}
}

func TestPostalCodeLookupMapsErrorsSafely(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantLog    string
	}{
		{name: "invalid", err: customers.ErrInvalidDetails, wantStatus: http.StatusBadRequest},
		{name: "not found", err: customers.ErrPostalCodeNotFound, wantStatus: http.StatusNotFound},
		{
			name:       "external status",
			err:        &customers.PostalCodeLookupError{Reason: "status", StatusCode: http.StatusInternalServerError},
			wantStatus: http.StatusServiceUnavailable,
			wantLog:    "cep lookup failed reason=status status=500",
		},
		{
			name:       "timeout",
			err:        &customers.PostalCodeLookupError{Reason: "timeout"},
			wantStatus: http.StatusServiceUnavailable,
			wantLog:    "cep lookup failed reason=timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := &fakePostalCodeLookup{err: tt.err}
			req := httptest.NewRequest(http.MethodGet, "/api/cep/01001000", nil)
			rec := httptest.NewRecorder()
			logs := captureServerLogs(t, func() {
				newTestHandlerWithPostalCode(t, lookup).ServeHTTP(rec, req)
			})

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantLog != "" && !strings.Contains(logs, tt.wantLog) {
				t.Fatalf("expected log %q, got %q", tt.wantLog, logs)
			}
			if strings.Contains(logs, "01001000") {
				t.Fatal("expected logs not to include CEP")
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("expected JSON body, got %v", err)
			}
			if len(body) != 4 {
				t.Fatalf("expected address response shape on errors too, got %#v", body)
			}
		})
	}
}

func TestPostalCodeLookupNilServiceReturnsServiceUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/cep/01001000", nil)
	rec := httptest.NewRecorder()
	logs := captureServerLogs(t, func() {
		newTestHandlerWithPostalCode(t, nil).ServeHTTP(rec, req)
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if !strings.Contains(logs, "cep lookup failed reason=not_configured") {
		t.Fatalf("expected safe not_configured log, got %q", logs)
	}
}

func newTestHandlerWithPostalCode(t *testing.T, lookup postalCodeLookupService) http.Handler {
	t.Helper()

	db, err := database.New(context.Background(), database.Config{
		MaxConns: config.DefaultDBMaxConns,
	})
	if err != nil {
		t.Fatalf("expected test database config to be valid, got %v", err)
	}
	t.Cleanup(db.Close)

	return newHandlerWithServices(db, nil, nil, nil, nil, nil, lookup, "")
}

func captureServerLogs(t *testing.T, run func()) string {
	t.Helper()

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&buffer)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
	})

	run()

	return buffer.String()
}

type fakePostalCodeLookup struct {
	address customers.PostalCodeAddress
	err     error

	calls          int
	lastPostalCode string
}

func (s *fakePostalCodeLookup) Lookup(_ context.Context, postalCode string) (customers.PostalCodeAddress, error) {
	s.calls++
	s.lastPostalCode = postalCode
	if s.err != nil {
		return customers.PostalCodeAddress{}, s.err
	}

	return s.address, nil
}

var _ postalCodeLookupService = (*fakePostalCodeLookup)(nil)
