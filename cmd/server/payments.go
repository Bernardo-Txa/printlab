package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const maxInfinitePayWebhookBodyBytes = 64 << 10

type paymentService interface {
	Available() bool
	StartCheckout(ctx context.Context, orderID string) (paymentsdomain.CheckoutStartResult, error)
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
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload invalido")
			return
		}

		var payload infinitePayWebhookRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxInfinitePayWebhookBodyBytes))
		if err := decoder.Decode(&payload); err != nil {
			if isMaxBytesError(err) {
				writePaymentWebhookJSON(w, http.StatusRequestEntityTooLarge, false, "Payload invalido")
				return
			}
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload invalido")
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			if isMaxBytesError(err) {
				writePaymentWebhookJSON(w, http.StatusRequestEntityTooLarge, false, "Payload invalido")
				return
			}
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload invalido")
			return
		}

		log.Print("payment webhook received")
		if service == nil || !service.Available() {
			log.Print("payment webhook unavailable")
			writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pagamento indisponivel")
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
			handlePaymentWebhookError(w, err, result)
			return
		}
		if result.Status != paymentsdomain.ReturnStatusConfirmed {
			handlePaymentWebhookError(w, paymentsdomain.ErrPaymentNotConfirmed, result)
			return
		}

		if result.Message == paymentsdomain.ReturnMessageAlreadyPaid {
			log.Printf("payment webhook already_paid order_id=%s", result.OrderID)
		} else {
			log.Printf("payment webhook verified order_id=%s", result.OrderID)
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
			log.Printf("payment checkout amount mismatch order_id=%s", requestedOrderID)
		} else {
			log.Printf("payment checkout unavailable%s", paymentProviderLogSuffix(err))
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
			log.Print("payment confirmation amount mismatch")
		} else {
			log.Printf("payment confirmation unavailable%s", paymentProviderLogSuffix(err))
		}
		renderHTML(w, r, http.StatusServiceUnavailable, templates.PaymentReturn(page))
	}
}

func handlePaymentWebhookError(w http.ResponseWriter, err error, result paymentsdomain.ReturnResult) {
	switch {
	case errors.Is(err, paymentsdomain.ErrInvalidReturn):
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Payload invalido")
	case errors.Is(err, paymentsdomain.ErrPaymentNotFound),
		errors.Is(err, paymentsdomain.ErrOrderNotFound):
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pedido nao encontrado")
	case errors.Is(err, paymentsdomain.ErrPaymentNotConfirmed):
		log.Printf("payment webhook pending order_id=%s", result.OrderID)
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Pagamento ainda nao confirmado")
	default:
		if !errors.Is(err, paymentsdomain.ErrAmountMismatch) {
			log.Printf("payment webhook unavailable%s", paymentProviderLogSuffix(err))
		}
		writePaymentWebhookJSON(w, http.StatusBadRequest, false, "Nao foi possivel confirmar o pagamento agora")
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

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func methodNotAllowedHandler(allowedMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allowedMethod)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
