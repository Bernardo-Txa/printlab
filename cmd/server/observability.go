package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

type requestIDContextKey struct{}

type operationalLogLevel string

const (
	operationalLogLevelInfo    operationalLogLevel = "info"
	operationalLogLevelWarning operationalLogLevel = "warning"
	operationalLogLevelError   operationalLogLevel = "error"
)

var operationalStandardLogger = log.New(os.Stdout, "", log.LstdFlags)

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

// logOperationalEvent emits only an event name, an internal safe level,
// categories and request correlation. Callers must never pass customer, order,
// cart or credential data. Errors use stderr; successful and expected events
// use stdout so runtimes can classify them without treating them as errors.
func logOperationalEvent(ctx context.Context, level operationalLogLevel, event string, fields string) {
	requestID := requestIDFromContext(ctx)
	message := "event=" + event + " level=" + string(level)
	if fields == "" {
		message += " request_id=" + requestID
	} else {
		message += " " + fields + " request_id=" + requestID
	}

	switch level {
	case operationalLogLevelInfo, operationalLogLevelWarning:
		operationalStandardLogger.Print(message)
	case operationalLogLevelError:
		log.Print(message)
	default:
		log.Print("event=observability_invalid_level level=error reason=invalid_level request_id=" + requestID)
	}
}
