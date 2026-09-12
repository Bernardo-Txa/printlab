package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	ProviderOperationCreateCheckout = "create_checkout"
	ProviderOperationPaymentCheck   = "payment_check"

	ProviderCategoryNetworkError       = "network_error"
	ProviderCategoryTimeout            = "timeout"
	ProviderCategoryHTTP400            = "http_400"
	ProviderCategoryHTTP401            = "http_401"
	ProviderCategoryHTTP403            = "http_403"
	ProviderCategoryHTTP404            = "http_404"
	ProviderCategoryHTTP409            = "http_409"
	ProviderCategoryHTTP422            = "http_422"
	ProviderCategoryHTTP429            = "http_429"
	ProviderCategoryHTTP5xx            = "http_5xx"
	ProviderCategoryInvalidJSON        = "invalid_json"
	ProviderCategoryInvalidCheckoutURL = "invalid_checkout_url"
	ProviderCategoryUnknown            = "unknown"

	maxProviderErrorBodyBytes      = 4096
	maxProviderDiagnosticCodeBytes = 64
	maxProviderDiagnosticTextBytes = 120
)

type ProviderError struct {
	Provider   string
	Operation  string
	StatusCode int
	Category   string
	Code       string
	Message    string
	Host       string
	cause      error
}

type ProviderErrorDetails struct {
	Provider   string
	Operation  string
	StatusCode int
	Category   string
	Code       string
	Message    string
	Host       string
}

func NewProviderError(operation string, category string, statusCode int, cause error) *ProviderError {
	if cause == nil {
		cause = ErrProviderUnavailable
	}

	return &ProviderError{
		Provider:   ProviderInfinitePay,
		Operation:  normalizeProviderOperation(operation),
		StatusCode: statusCode,
		Category:   normalizeProviderCategory(category),
		cause:      cause,
	}
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "payment provider error"
	}

	var builder strings.Builder
	builder.WriteString("payment provider error")
	builder.WriteString(" provider=")
	builder.WriteString(safeLogToken(e.Provider))
	builder.WriteString(" operation=")
	builder.WriteString(safeLogToken(e.Operation))
	if e.StatusCode > 0 {
		fmt.Fprintf(&builder, " status=%d", e.StatusCode)
	}
	builder.WriteString(" category=")
	builder.WriteString(safeLogToken(e.Category))
	if host := safeLogToken(e.Host); host != "" {
		builder.WriteString(" host=")
		builder.WriteString(host)
	}

	return builder.String()
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func ProviderErrorDetailsFor(err error) (ProviderErrorDetails, bool) {
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr == nil {
		return ProviderErrorDetails{}, false
	}

	return ProviderErrorDetails{
		Provider:   providerErr.Provider,
		Operation:  providerErr.Operation,
		StatusCode: providerErr.StatusCode,
		Category:   providerErr.Category,
		Code:       providerErr.Code,
		Message:    providerErr.Message,
		Host:       providerErr.Host,
	}, true
}

func ProviderLogFields(err error) string {
	details, ok := ProviderErrorDetailsFor(err)
	if !ok {
		return ""
	}

	fields := []string{
		"provider=" + safeLogToken(details.Provider),
		"operation=" + safeLogToken(details.Operation),
	}
	if details.StatusCode > 0 {
		fields = append(fields, fmt.Sprintf("status=%d", details.StatusCode))
	}
	fields = append(fields, "category="+safeLogToken(details.Category))
	if host := safeLogToken(details.Host); host != "" {
		fields = append(fields, "host="+host)
	}
	if code := safeLogToken(details.Code); code != "" {
		fields = append(fields, "code="+code)
	}

	return strings.Join(fields, " ")
}

func providerHTTPError(operation string, statusCode int, body io.Reader) *ProviderError {
	err := NewProviderError(operation, providerCategoryForHTTPStatus(statusCode), statusCode, ErrProviderUnavailable)
	diagnostic := providerErrorBodyDiagnostic(body)
	err.Code = diagnostic.Code
	err.Message = diagnostic.Message
	return err
}

func providerNetworkError(operation string, err error) *ProviderError {
	category := ProviderCategoryNetworkError
	if providerErrorIsTimeout(err) {
		category = ProviderCategoryTimeout
	}

	return NewProviderError(operation, category, 0, ErrProviderUnavailable)
}

func providerInvalidJSONError(operation string) *ProviderError {
	return NewProviderError(operation, ProviderCategoryInvalidJSON, 0, ErrProviderUnavailable)
}

func providerInvalidCheckoutURLError(rawURL string) *ProviderError {
	err := NewProviderError(ProviderOperationCreateCheckout, ProviderCategoryInvalidCheckoutURL, 0, ErrInvalidCheckoutURL)
	err.Host = safeURLHost(rawURL)
	return err
}

func providerCategoryForHTTPStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return ProviderCategoryHTTP400
	case http.StatusUnauthorized:
		return ProviderCategoryHTTP401
	case http.StatusForbidden:
		return ProviderCategoryHTTP403
	case http.StatusNotFound:
		return ProviderCategoryHTTP404
	case http.StatusConflict:
		return ProviderCategoryHTTP409
	case http.StatusUnprocessableEntity:
		return ProviderCategoryHTTP422
	case http.StatusTooManyRequests:
		return ProviderCategoryHTTP429
	default:
		if statusCode >= http.StatusInternalServerError && statusCode <= 599 {
			return ProviderCategoryHTTP5xx
		}
		return ProviderCategoryUnknown
	}
}

func providerErrorIsTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

type providerBodyDiagnostic struct {
	Code    string
	Message string
}

func providerErrorBodyDiagnostic(body io.Reader) providerBodyDiagnostic {
	if body == nil {
		return providerBodyDiagnostic{}
	}

	payload, err := io.ReadAll(io.LimitReader(body, maxProviderErrorBodyBytes))
	if err != nil || len(payload) == 0 {
		return providerBodyDiagnostic{}
	}

	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return providerBodyDiagnostic{}
	}

	return providerBodyDiagnostic{
		Code:    sanitizeProviderDiagnostic(providerStringField(document, "code"), maxProviderDiagnosticCodeBytes),
		Message: sanitizeProviderDiagnostic(firstProviderStringField(document, "message", "error"), maxProviderDiagnosticTextBytes),
	}
}

func providerStringField(document map[string]any, key string) string {
	value, ok := document[key]
	if !ok {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}

func firstProviderStringField(document map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := providerStringField(document, key); value != "" {
			return value
		}
	}

	return ""
}

func sanitizeProviderDiagnostic(value string, maxBytes int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return ""
	}
	if providerDiagnosticLooksSensitive(value) {
		return ""
	}
	if len(value) > maxBytes {
		value = value[:maxBytes]
	}

	return value
}

func providerDiagnosticLooksSensitive(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(value, "@") ||
		strings.Contains(value, "+") ||
		strings.Contains(lower, "http://") ||
		strings.Contains(lower, "https://") ||
		strings.Contains(lower, "checkout.") ||
		strings.Contains(lower, "transaction_nsu") ||
		strings.Contains(lower, "order_nsu") ||
		strings.Contains(lower, "customer") ||
		strings.Contains(lower, "email") ||
		strings.Contains(lower, "phone") ||
		strings.Contains(lower, "telefone") ||
		strings.Contains(lower, "address") ||
		strings.Contains(lower, "endereco") ||
		strings.Contains(lower, "cpf") ||
		strings.Contains(lower, "cep") {
		return true
	}

	var consecutiveDigits int
	for _, char := range value {
		if char < 32 || char > 126 {
			return true
		}
		if char >= '0' && char <= '9' {
			consecutiveDigits++
			if consecutiveDigits >= 8 {
				return true
			}
			continue
		}
		consecutiveDigits = 0
	}

	return false
}

func safeURLHost(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil {
		return ""
	}

	return safeLogToken(parsed.Host)
}

func safeLogToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '_' || char == '-' || char == '.':
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func normalizeProviderOperation(operation string) string {
	switch strings.TrimSpace(operation) {
	case ProviderOperationCreateCheckout:
		return ProviderOperationCreateCheckout
	case ProviderOperationPaymentCheck:
		return ProviderOperationPaymentCheck
	default:
		return ProviderCategoryUnknown
	}
}

func normalizeProviderCategory(category string) string {
	switch strings.TrimSpace(category) {
	case ProviderCategoryNetworkError,
		ProviderCategoryTimeout,
		ProviderCategoryHTTP400,
		ProviderCategoryHTTP401,
		ProviderCategoryHTTP403,
		ProviderCategoryHTTP404,
		ProviderCategoryHTTP409,
		ProviderCategoryHTTP422,
		ProviderCategoryHTTP429,
		ProviderCategoryHTTP5xx,
		ProviderCategoryInvalidJSON,
		ProviderCategoryInvalidCheckoutURL,
		ProviderCategoryUnknown:
		return strings.TrimSpace(category)
	default:
		return ProviderCategoryUnknown
	}
}
