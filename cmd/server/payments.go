package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type paymentService interface {
	Available() bool
	StartCheckout(ctx context.Context, orderID string) (paymentsdomain.CheckoutStartResult, error)
	ConfirmReturn(ctx context.Context, input paymentsdomain.ReturnInput) (paymentsdomain.ReturnResult, error)
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
			log.Print("payment checkout unavailable")
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
			log.Print("payment confirmation unavailable")
		}
		renderHTML(w, r, http.StatusServiceUnavailable, templates.PaymentReturn(page))
	}
}
