package admin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testAdminUserID = "11111111-1111-1111-1111-111111111111"

func TestSupabaseAuthClientSignsInWithEmailPassword(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/auth/v1/token" {
			t.Fatalf("expected token path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("grant_type") != "password" {
			t.Fatalf("expected password grant, got %q", r.URL.RawQuery)
		}
		if got := r.Header.Get("apikey"); got != "sb_publishable_test" {
			t.Fatalf("expected publishable key header, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("expected JSON content type, got %q", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ignored","refresh_token":"ignored","user":{"id":"` + testAdminUserID + `"}}`))
	}))
	defer server.Close()

	client := newTestAuthClient(t, server.URL, server.Client())
	user, err := client.SignInWithPassword(context.Background(), " admin@example.com ", "correct-password")
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if !sawRequest {
		t.Fatal("expected auth request")
	}
	if user.ID != testAdminUserID {
		t.Fatalf("expected user id %q, got %q", testAdminUserID, user.ID)
	}
}

func TestSupabaseAuthClientHandlesHTTPFailuresSafely(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: http.StatusBadRequest, want: ErrAuthRejected},
		{status: http.StatusTooManyRequests, want: ErrAuthUnavailable},
		{status: http.StatusInternalServerError, want: ErrAuthUnavailable},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"message":"invalid"}`, tt.status)
			}))
			defer server.Close()

			client := newTestAuthClient(t, server.URL, server.Client())
			_, err := client.SignInWithPassword(context.Background(), "admin@example.com", "secret-password")
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			assertErrorDoesNotContainSensitiveAuthData(t, err)
		})
	}
}

func TestSupabaseAuthClientHandlesTimeoutSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := newTestAuthClient(t, server.URL, &http.Client{Timeout: 10 * time.Millisecond})
	_, err := client.SignInWithPassword(context.Background(), "admin@example.com", "secret-password")
	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("expected auth unavailable, got %v", err)
	}
	assertErrorDoesNotContainSensitiveAuthData(t, err)
}

func TestSupabaseAuthClientHandlesNetworkErrorSafely(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("expected listener, got %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	client := newTestAuthClient(t, "http://"+addr, &http.Client{Timeout: time.Second})
	_, err = client.SignInWithPassword(context.Background(), "admin@example.com", "secret-password")
	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("expected auth unavailable, got %v", err)
	}
	assertErrorDoesNotContainSensitiveAuthData(t, err)
}

func TestSupabaseAuthClientRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `not-json`},
		{name: "missing user", body: `{}`},
		{name: "invalid user UUID", body: `{"user":{"id":"not-a-uuid"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestAuthClient(t, server.URL, server.Client())
			_, err := client.SignInWithPassword(context.Background(), "admin@example.com", "secret-password")
			if !errors.Is(err, ErrAuthUnavailable) {
				t.Fatalf("expected auth unavailable, got %v", err)
			}
			assertErrorDoesNotContainSensitiveAuthData(t, err)
		})
	}
}

func newTestAuthClient(t *testing.T, baseURL string, httpClient *http.Client) *SupabaseAuthClient {
	t.Helper()

	client, err := NewSupabaseAuthClient(SupabaseAuthClientConfig{
		SupabaseURL:    baseURL,
		PublishableKey: "sb_publishable_test",
		HTTPClient:     httpClient,
	})
	if err != nil {
		t.Fatalf("expected auth client, got %v", err)
	}

	return client
}

func assertErrorDoesNotContainSensitiveAuthData(t *testing.T, err error) {
	t.Helper()

	text := err.Error()
	for _, forbidden := range []string{"secret-password", "admin@example.com", "sb_publishable_test"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected error not to contain %q, got %q", forbidden, text)
		}
	}
}
