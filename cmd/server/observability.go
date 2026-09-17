package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
)

type requestIDContextKey struct{}

func requestIDContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return requestID
	}
	return ""
}

func newRequestID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	// A failed entropy read is extraordinarily unusual. Keep the value opaque and
	// avoid accepting a caller-controlled correlation identifier.
	return "unavailable"
}

// logOperationalEvent emits only an event name, safe categories and request
// correlation. Callers must never pass customer, order, cart or credential data.
func logOperationalEvent(ctx context.Context, event string, fields string) {
	requestID := requestIDFromContext(ctx)
	if fields == "" {
		log.Printf("event=%s level=error request_id=%s", event, requestID)
		return
	}
	log.Printf("event=%s level=error %s request_id=%s", event, fields, requestID)
}
