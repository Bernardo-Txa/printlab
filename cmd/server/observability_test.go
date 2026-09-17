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
	previousErrorOutput := log.Writer()
	previousInfoOutput := operationalStandardLogger.Writer()
	var errorOutput, infoOutput strings.Builder
	log.SetOutput(&errorOutput)
	operationalStandardLogger.SetOutput(&infoOutput)
	t.Cleanup(func() {
		log.SetOutput(previousErrorOutput)
		operationalStandardLogger.SetOutput(previousInfoOutput)
	})

	logOperationalEvent(requestIDContext(context.Background(), "0123456789abcdef"), operationalLogLevelError, "payment_check_failed", "reason=provider_unavailable category=timeout")
	if got := errorOutput.String(); !strings.Contains(got, "event=payment_check_failed level=error reason=provider_unavailable category=timeout request_id=0123456789abcdef") {
		t.Fatalf("unexpected log output: %q", got)
	}
	if got := infoOutput.String(); got != "" {
		t.Fatalf("error event written to stdout: %q", got)
	}
	logOperationalEvent(requestIDContext(context.Background(), "0123456789abcdef"), operationalLogLevelInfo, "payment_webhook_processed", "reason=confirmed")
	if got := infoOutput.String(); !strings.Contains(got, "event=payment_webhook_processed level=info reason=confirmed request_id=0123456789abcdef") {
		t.Fatalf("unexpected stdout event: %q", got)
	}
}

func captureOperationalLogs(t *testing.T) (*strings.Builder, *strings.Builder) {
	t.Helper()
	previousErrorOutput := log.Writer()
	previousInfoOutput := operationalStandardLogger.Writer()
	var errorOutput, standardOutput strings.Builder
	log.SetOutput(&errorOutput)
	operationalStandardLogger.SetOutput(&standardOutput)
	t.Cleanup(func() {
		log.SetOutput(previousErrorOutput)
		operationalStandardLogger.SetOutput(previousInfoOutput)
	})
	return &errorOutput, &standardOutput
}
