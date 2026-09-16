package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSupabaseMFAContracts(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer test-access" || r.Header.Get("apikey") != "sb_publishable_test" {
			t.Error("MFA must use user bearer and publishable key")
		}
		var payload map[string]string
		if r.Method == http.MethodPost && json.NewDecoder(r.Body).Decode(&payload) != nil {
			t.Error("expected JSON body")
		}
		switch r.URL.Path {
		case "/auth/v1/user":
			if r.Method != "GET" {
				t.Error("user requires GET")
			}
			_, _ = fmt.Fprintf(w, `{"id":%q,"factors":[{"id":%q,"factor_type":"totp","status":"verified","friendly_name":"backup"}]}`, testAdminUserID, testFactorID)
		case "/auth/v1/factors":
			if payload["factor_type"] != "totp" || payload["friendly_name"] != "PrintLab Admin" || payload["issuer"] != "PrintLab" {
				t.Error("wrong enrollment contract")
			}
			_, _ = fmt.Fprintf(w, `{"id":%q,"type":"totp","totp":{"qr_code":"<svg></svg>","secret":"test-secret","uri":"otpauth://ignored"}}`, testFactorID)
		case "/auth/v1/factors/" + testFactorID + "/challenge":
			_, _ = fmt.Fprintf(w, `{"id":%q,"expires_at":9999999999}`, testFactorID)
		case "/auth/v1/factors/" + testFactorID + "/verify":
			if payload["challenge_id"] != testFactorID || payload["code"] != "123456" {
				t.Error("wrong verify contract")
			}
			_, _ = fmt.Fprintf(w, `{"access_token":"updated-access","refresh_token":"must-discard","user":{"id":%q}}`, testAdminUserID)
		case "/auth/v1/factors/" + testFactorID:
			if r.Method != "DELETE" {
				t.Error("cleanup requires DELETE")
			}
			_, _ = fmt.Fprintf(w, `{"id":%q}`, testFactorID)
		default:
			t.Error("unexpected endpoint")
		}
	}))
	defer server.Close()
	c := newTestAuthClient(t, server.URL, server.Client())
	ctx := context.Background()
	u, err := c.GetUser(ctx, "test-access")
	if err != nil || u.ID != testAdminUserID || len(u.Factors) != 1 || u.Factors[0].FriendlyName != "backup" {
		t.Fatal("user factors not decoded")
	}
	e, err := c.EnrollTOTP(ctx, "test-access")
	if err != nil || e.TOTP.Secret != "test-secret" || e.TOTP.QRCode != "<svg></svg>" {
		t.Fatal("enrollment not decoded")
	}
	id, err := c.Challenge(ctx, "test-access", testFactorID)
	if err != nil || id != testFactorID {
		t.Fatal("challenge not decoded")
	}
	s, err := c.Verify(ctx, "test-access", testFactorID, id, "123456")
	if err != nil || s.AccessToken != "updated-access" {
		t.Fatal("verify token not decoded")
	}
	encoded, _ := json.Marshal(s)
	if strings.Contains(string(encoded), "refresh") || strings.Contains(string(encoded), "must-discard") {
		t.Fatal("refresh token retained")
	}
	if err := c.UnenrollFactor(ctx, "test-access", testFactorID); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 5 {
		t.Fatal("unexpected extra request")
	}
}

func TestSupabaseAuthClientValidatesQRCodeStructure(t *testing.T) {
	cases := []struct {
		name    string
		qr      string
		secret  string
		typ     string
		wantErr bool
	}{
		{name: "simple SVG", qr: "<svg></svg>", secret: "secret", typ: "totp"},
		{name: "XML declaration", qr: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>", secret: "secret", typ: "totp"},
		{name: "leading whitespace", qr: "\n  <svg xmlns=\"http://www.w3.org/2000/svg\">\n  </svg>\n", secret: "secret", typ: "totp"},
		{name: "no SVG", qr: "invalid", secret: "secret", typ: "totp", wantErr: true},
		{name: "unclosed SVG", qr: "<svg>", secret: "secret", typ: "totp", wantErr: true},
		{name: "empty secret", qr: "<svg></svg>", typ: "totp", wantErr: true},
		{name: "wrong factor type", qr: "<svg></svg>", secret: "secret", typ: "phone", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/v1/factors" {
					t.Fatalf("expected enrollment endpoint, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"id":%q,"type":%q,"totp":{"qr_code":%q,"secret":%q}}`, testFactorID, tc.typ, tc.qr, tc.secret)
			}))
			defer server.Close()

			client := newTestAuthClient(t, server.URL, server.Client())
			_, err := client.EnrollTOTP(context.Background(), "test-access")
			if tc.wantErr {
				if !errors.Is(err, ErrAuthInvalidResponse) {
					t.Fatalf("expected invalid response, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected valid enrollment, got %v", err)
			}
		})
	}
}

func TestSupabaseAuthResponseLimitsAndStatuses(t *testing.T) {
	operations := []struct {
		name string
		call func(*SupabaseAuthClient) error
	}{
		{"password", func(c *SupabaseAuthClient) error {
			_, e := c.SignInWithPassword(context.Background(), "admin@example.com", "password")
			return e
		}},
		{"user", func(c *SupabaseAuthClient) error { _, e := c.GetUser(context.Background(), "token"); return e }},
		{"enroll", func(c *SupabaseAuthClient) error { _, e := c.EnrollTOTP(context.Background(), "token"); return e }},
		{"challenge", func(c *SupabaseAuthClient) error {
			_, e := c.Challenge(context.Background(), "token", testFactorID)
			return e
		}},
		{"verify", func(c *SupabaseAuthClient) error {
			_, e := c.Verify(context.Background(), "token", testFactorID, testFactorID, "123456")
			return e
		}},
		{"cleanup", func(c *SupabaseAuthClient) error {
			return c.UnenrollFactor(context.Background(), "token", testFactorID)
		}},
	}
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"401", 401, `{"message":"private-body"}`, ErrAuthRejected},
		{"422", 422, `{"message":"private-body"}`, ErrAuthRejected},
		{"429", 429, `{"message":"private-body"}`, ErrAuthRateLimited},
		{"503", 503, `{"message":"private-body"}`, ErrAuthUnavailable},
		{"malformed", 200, `{"invalid"`, ErrAuthInvalidResponse},
		{"oversized", 200, `{}` + strings.Repeat(" ", maxAuthResponseBytes), ErrAuthInvalidResponse},
		{"second JSON", 200, `{} {}`, ErrAuthInvalidResponse},
		{"missing fields", 200, `{}`, ErrAuthInvalidResponse},
	}
	for _, op := range operations {
		for _, tt := range cases {
			t.Run(op.name+"/"+tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				}))
				defer server.Close()
				err := op.call(newTestAuthClient(t, server.URL, server.Client()))
				want := tt.want
				if want == ErrAuthRejected && op.name == "user" {
					want = ErrUnauthenticated
				}
				if want == ErrAuthRejected && op.name == "verify" {
					want = ErrMFAInvalidCode
				}
				if !errors.Is(err, want) {
					t.Fatalf("got %v, want %v", err, want)
				}
				if strings.Contains(err.Error(), "private-body") {
					t.Fatal("provider body leaked")
				}
			})
		}
	}
}

func TestSupabaseMFARequestTimeoutAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(30 * time.Millisecond) }))
	client := newTestAuthClient(t, server.URL, &http.Client{Timeout: 10 * time.Millisecond})
	_, err := client.GetUser(context.Background(), "token")
	server.Close()
	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatal("timeout must be safe")
	}
	var followed bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	_, err = newTestAuthClient(t, redirect.URL, redirect.Client()).GetUser(context.Background(), "token")
	if !errors.Is(err, ErrAuthUnavailable) || followed {
		t.Fatal("auth request must not follow redirects")
	}
}
