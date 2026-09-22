package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const maxInfinitePayWebhookBodyBytes = 64 << 10

type paymentService interface {
	Available() bool
	StartCheckout(ctx context.Context, orderID string) (paymentsdomain.CheckoutStartResult, error)
	StartCheckoutForCustomer(ctx context.Context, orderID string, customerAuthUserID string) (paymentsdomain.CheckoutStartResult, error)
	ConfirmReturn(ctx context.Context, input paymentsdomain.ReturnInput) (paymentsdomain.ReturnResult, error)
	ConfirmWebhook(ctx context.Context, input paymentsdomain.WebhookInput) (paymentsdomain.ReturnResult, error)
}

func startPaymentHandler(service paymentService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		orderID := r.PathValue("id")
		if !paymentsdomain.ValidOrderID(orderID) {
			http.NotFound(w, r)
			return
		}
		setCheckoutPrivateCache(w)

		if service == nil || !service.Available() {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_checkout_unavailable", "reason=service_unavailable")
			http.Redirect(w, r, "/pedido/"+orderID+"?pagamento=indisponivel", http.StatusSeeOther)
			return
		}

		result, err := service.StartCheckout(r.Context(), orderID)
		if err != nil {
			handleStartPaymentError(w, r, err, result.OrderID, orderID)
			return
		}
		if err := paymentsdomain.ValidateCheckoutURL(result.CheckoutURL); err != nil {
			handleStartPaymentError(w, r, err, result.OrderID, orderID)
			return
		}

		http.Redirect(w, r, result.CheckoutURL, http.StatusSeeOther)
	}
}

func paymentReturnHandler(service paymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCheckoutPrivateCache(w)
		if service == nil || !service.Available() {
			page := paymentsdomain.ReturnPageFor(paymentsdomain.ReturnResult{Status: paymentsdomain.ReturnStatusUnavailable})
			renderHTML(w, r, http.StatusServiceUnavailable, templates.PaymentReturn(page))
			return
		}

		result, err := service.ConfirmReturn(r.Context(), paymentsdomain.ReturnInput{
			OrderNSU:       r.URL.Query().Get("order_nsu"),
			TransactionNSU: r.URL.Query().Get("transaction_nsu"),
			Slug:           r.URL.Query().Get("slug"),
		})
		if err != nil {
			handlePaymentReturnError(w, r, err, result)
			return
		}
		if result.Status == paymentsdomain.ReturnStatusConfirmed && paymentsdomain.ValidOrderID(result.OrderID) {
			http.Redirect(w, r, "/pedido/"+result.OrderID+"?pagamento=confirmado", http.StatusSeeOther)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.PaymentReturn(paymentsdomain.ReturnPageFor(result)))
	}
}

func infinitePayWebhookHandler(service paymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload inválido")
			return
		}

		var payload infinitePayWebhookRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxInfinitePayWebhookBodyBytes))
		if err := decoder.Decode(&payload); err != nil {
			if isMaxBytesError(err) {
				writePaymentWebhookJSON(w, http.StatusRequestEntityTooLarge, false, "Payload inválido")
				return
			}
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload inválido")
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			if isMaxBytesError(err) {
				writePaymentWebhookJSON(w, http.StatusRequestEntityTooLarge, false, "Payload inválido")
				return
			}
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload inválido")
			return
		}

		if service == nil || !service.Available() {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_webhook_unavailable", "reason=service_unavailable")
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pagamento indisponível")
			return
		}

		result, err := service.ConfirmWebhook(r.Context(), paymentsdomain.WebhookInput{
			OrderNSU:        payload.OrderNSU,
			TransactionNSU:  payload.TransactionNSU,
			InvoiceSlug:     payload.InvoiceSlug,
			AmountCents:     payload.Amount,
			PaidAmountCents: payload.PaidAmount,
			Installments:    payload.Installments,
			CaptureMethod:   payload.CaptureMethod,
		})
		if err != nil {
			handlePaymentWebhookError(w, r.Context(), err, result)
			return
		}
		if result.Status != paymentsdomain.ReturnStatusConfirmed {
			handlePaymentWebhookError(w, r.Context(), paymentsdomain.ErrPaymentNotConfirmed, result)
			return
		}

		if result.Message == paymentsdomain.ReturnMessageAlreadyPaid {
			logOperationalEvent(r.Context(), operationalLogLevelInfo, "payment_webhook_processed", "reason=already_paid")
		} else {
			logOperationalEvent(r.Context(), operationalLogLevelInfo, "payment_webhook_processed", "reason=confirmed")
		}
		writePaymentWebhookJSON(w, http.StatusOK, true, "")
	}
}

func handleStartPaymentError(w http.ResponseWriter, r *http.Request, err error, resultOrderID string, requestedOrderID string) {
	switch {
	case errors.Is(err, paymentsdomain.ErrInvalidOrderID),
		errors.Is(err, paymentsdomain.ErrOrderNotFound):
		http.NotFound(w, r)
	case errors.Is(err, paymentsdomain.ErrOrderAlreadyPaid):
		redirectOrderID := resultOrderID
		if redirectOrderID == "" {
			redirectOrderID = requestedOrderID
		}
		http.Redirect(w, r, "/pedido/"+redirectOrderID, http.StatusSeeOther)
	default:
		if errors.Is(err, paymentsdomain.ErrAmountMismatch) {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_checkout_unavailable", "reason=amount_mismatch")
		} else {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_checkout_unavailable", "reason=provider_unavailable"+paymentProviderLogSuffix(err))
		}
		http.Redirect(w, r, "/pedido/"+requestedOrderID+"?pagamento=indisponivel", http.StatusSeeOther)
	}
}

func handlePaymentReturnError(w http.ResponseWriter, r *http.Request, err error, result paymentsdomain.ReturnResult) {
	page := paymentsdomain.ReturnPageFor(result)
	switch {
	case errors.Is(err, paymentsdomain.ErrInvalidReturn):
		renderHTML(w, r, http.StatusBadRequest, templates.PaymentReturn(page))
	case errors.Is(err, paymentsdomain.ErrPaymentNotFound),
		errors.Is(err, paymentsdomain.ErrOrderNotFound):
		http.NotFound(w, r)
	default:
		if errors.Is(err, paymentsdomain.ErrAmountMismatch) {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_check_failed", "reason=amount_mismatch")
		} else {
			logOperationalEvent(r.Context(), operationalLogLevelError, "payment_check_failed", "reason=provider_unavailable"+paymentProviderLogSuffix(err))
		}
		renderHTML(w, r, http.StatusServiceUnavailable, templates.PaymentReturn(page))
	}
}

func handlePaymentWebhookError(w http.ResponseWriter, ctx context.Context, err error, result paymentsdomain.ReturnResult) {
	switch {
	case errors.Is(err, paymentsdomain.ErrInvalidReturn):
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload inválido")
	case errors.Is(err, paymentsdomain.ErrPaymentNotFound),
		errors.Is(err, paymentsdomain.ErrOrderNotFound):
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pedido não encontrado")
	case errors.Is(err, paymentsdomain.ErrPaymentNotConfirmed):
		logOperationalEvent(ctx, operationalLogLevelWarning, "payment_webhook_invalid", "reason=not_confirmed")
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pagamento ainda não confirmado")
	case errors.Is(err, paymentsdomain.ErrAmountMismatch):
		logOperationalEvent(ctx, operationalLogLevelError, "payment_webhook_processing_failed", "reason=amount_mismatch")
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Não foi possível confirmar o pagamento agora")
	default:
		logOperationalEvent(ctx, operationalLogLevelError, "payment_webhook_processing_failed", "reason=provider_unavailable"+paymentProviderLogSuffix(err))
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Não foi possível confirmar o pagamento agora")
	}
}

func paymentProviderLogSuffix(err error) string {
	fields := paymentsdomain.ProviderLogFields(err)
	if fields == "" {
		return ""
	}

	return " " + fields
}

type infinitePayWebhookRequest struct {
	InvoiceSlug    string `json:"invoice_slug"`
	TransactionNSU string `json:"transaction_nsu"`
	OrderNSU       string `json:"order_nsu"`
	Amount         *int64 `json:"amount"`
	PaidAmount     *int64 `json:"paid_amount"`
	Installments   *int   `json:"installments"`
	CaptureMethod  string `json:"capture_method"`
}

type paymentWebhookResponse struct {
	Success bool    `json:"success"`
	Message *string `json:"message"`
}

func writePaymentWebhookJSON(w http.ResponseWriter, status int, success bool, message string) {
	var responseMessage *string
	if message != "" {
		responseMessage = &message
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(paymentWebhookResponse{
		Success: success,
		Message: responseMessage,
	})
}

func isJSONContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return mediaType == "application/json"
}

func methodNotAllowedHandler(allowedMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allowedMethod)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
