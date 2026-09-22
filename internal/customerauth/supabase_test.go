package customerauth

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSupabaseExchangeCodeUsesPKCEGrant(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/auth/v1/token" || r.URL.Query().Get("grant_type") != "pkce" {
			t.Fatalf("expected pkce token endpoint, got %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["auth_code"] != "auth-code" {
			t.Fatalf("expected auth_code, got %#v", payload)
		}
		return jsonResponse(http.StatusOK, `{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"user":{"id":"user-1","email":"cliente@example.com"}}`), nil
	})
	client, err := NewSupabaseClient(SupabaseClientConfig{SupabaseURL: "https://supabase.test", PublishableKey: "public-key", HTTPClient: &http.Client{Transport: transport}})
	if err != nil {
		t.Fatalf("client config: %v", err)
	}

	session, err := client.ExchangeCode(t.Context(), "auth-code")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if session.AccessToken != "access-token" || session.RefreshToken != "refresh-token" || session.User.Email != "cliente@example.com" {
		t.Fatalf("unexpected session %#v", session)
	}
}

func TestSupabaseRejectedLoginCanIdentifyUnconfirmedEmail(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error_code":"email_not_confirmed","msg":"Email not confirmed"}`), nil
	})
	client, err := NewSupabaseClient(SupabaseClientConfig{SupabaseURL: "https://supabase.test", PublishableKey: "public-key", HTTPClient: &http.Client{Transport: transport}})
	if err != nil {
		t.Fatalf("client config: %v", err)
	}

	_, err = client.SignInWithPassword(t.Context(), "cliente@example.com", "senha-segura")
	if err == nil || !strings.Contains(err.Error(), ErrEmailNotConfirmed.Error()) {
		t.Fatalf("expected ErrEmailNotConfirmed, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
