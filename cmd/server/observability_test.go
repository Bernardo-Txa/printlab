package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityMiddlewareAddsOpaqueRequestID(t *testing.T) {
	handler := securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestIDFromContext(r.Context()); len(got) != 16 {
			t.Fatalf("request ID length = %d, want 16", len(got))
		}
	}), "", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := rec.Header().Get("X-Request-ID"); len(got) != 16 {
		t.Fatalf("X-Request-ID = %q, want 16 opaque hexadecimal chars", got)
	}
}

func TestOperationalEventUsesOnlySafeStructuredFields(t *testing.T) {
	previous := log.Writer()
	var output strings.Builder
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })

	logOperationalEvent(requestIDContext(context.Background(), "0123456789abcdef"), "payment_check_failed", "reason=provider_unavailable category=timeout")
	if got := output.String(); !strings.Contains(got, "event=payment_check_failed level=error reason=provider_unavailable category=timeout request_id=0123456789abcdef") {
		t.Fatalf("unexpected log output: %q", got)
	}
}
