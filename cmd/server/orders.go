package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type orderReviewService interface {
	Review(ctx context.Context, tokenHash []byte, stale bool) (ordersdomain.ReviewPage, error)
	Confirm(ctx context.Context, tokenHash []byte, expectedFingerprint string) (ordersdomain.ConfirmResult, error)
	Get(ctx context.Context, orderID string) (ordersdomain.OrderPage, error)
	Track(ctx context.Context, trackingID string) (ordersdomain.TrackingPage, error)
}

func checkoutReviewPageHandler(service orderReviewService, cookies *cartdomain.CookieManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}
		setCheckoutPrivateCache(w)

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutReviewUnavailable())
			return
		}

		page, err := service.Review(r.Context(), tokenHash, r.URL.Query().Get("atualizado") == "1")
		if err != nil {
			handleOrderCheckoutError(w, r, err, page)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.CheckoutReview(page))
	}
}

func confirmOrderHandler(service orderReviewService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		_, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}
		setCheckoutPrivateCache(w)

		if service == nil {
			logOperationalEvent(r.Context(), operationalLogLevelError, "order_creation_failed", "reason=service_unavailable")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutReviewUnavailable())
			return
		}

		if err := r.ParseForm(); err != nil {
			writeInvalidBodyError(w, "invalid order request", err)
			return
		}

		fingerprint := strings.TrimSpace(r.PostFormValue("review_fingerprint"))
		var result ordersdomain.ConfirmResult
		var err error
		if customerService, ok := service.(interface {
			ConfirmForCustomer(context.Context, []byte, string, string) (ordersdomain.ConfirmResult, error)
		}); ok {
			id := ""
			if profile, profileOK := customerauth.ProfileFromContext(r.Context()); profileOK {
				id = profile.ID
			}
			result, err = customerService.ConfirmForCustomer(r.Context(), tokenHash, fingerprint, id)
		} else {
			result, err = service.Confirm(r.Context(), tokenHash, fingerprint)
		}
		if err != nil {
			if orderCreationOperationalFailure(err) {
				logOperationalEvent(r.Context(), operationalLogLevelError, "order_creation_failed", "reason=confirmation_failed")
			}
			handleOrderCheckoutError(w, r, err, result.Page)
			return
		}

		if result.ExpireCookie {
			ensureCartCookies(cookies).ExpireCookie(w)
		}
		http.Redirect(w, r, "/pedido/"+result.OrderID, http.StatusSeeOther)
	}
}

func orderCreationOperationalFailure(err error) bool {
	return !errors.Is(err, ordersdomain.ErrCartRequired) &&
		!errors.Is(err, ordersdomain.ErrEmptyCart) &&
		!errors.Is(err, ordersdomain.ErrUnavailableItems) &&
		!errors.Is(err, ordersdomain.ErrDetailsRequired) &&
		!errors.Is(err, ordersdomain.ErrShippingRequired) &&
		!errors.Is(err, ordersdomain.ErrShippingExpired) &&
		!errors.Is(err, ordersdomain.ErrShippingChanged) &&
		!errors.Is(err, ordersdomain.ErrStaleReview)
}

func orderPageHandler(service orderReviewService, payment paymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCheckoutPrivateCache(w)

		orderID := r.PathValue("id")
		if !ordersdomain.ValidOrderID(orderID) {
			http.NotFound(w, r)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.OrderUnavailable())
			return
		}

		page, err := service.Get(r.Context(), orderID)
		if err != nil {
			handleOrderPageError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.OrderConfirmation(page, orderPaymentView(page, payment, r.URL.Query().Get("pagamento"))))
	}
}

func orderTrackingPageHandler(service orderReviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setOrderTrackingHeaders(w)

		trackingID := r.PathValue("tracking_id")
		if !ordersdomain.ValidTrackingID(trackingID) {
			http.NotFound(w, r)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.OrderTrackingUnavailable())
			return
		}

		page, err := service.Track(r.Context(), trackingID)
		if err != nil {
			handleOrderTrackingPageError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.OrderTracking(page))
	}
}

func setOrderTrackingHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", checkoutPrivateCacheControl)
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func orderPaymentView(page ordersdomain.OrderPage, service paymentService, paymentQuery string) paymentsdomain.OrderPaymentView {
	view := paymentsdomain.OrderPaymentView{}
	if page.Status == ordersdomain.StatusPaid {
		view.ShowConfirmed = true
		view.Message = "Pagamento confirmado."
		return view
	}
	if paymentQuery == "indisponivel" {
		view.ShowUnavailable = true
		view.UnavailableMessage = "Pagamento temporariamente indisponível. Tente novamente em alguns instantes."
	}
	if page.Status != ordersdomain.StatusPendingPayment {
		return view
	}
	if service == nil || !service.Available() {
		view.ShowUnavailable = true
		view.UnavailableMessage = "Pagamento temporariamente indisponível. Tente novamente em alguns instantes."
		return view
	}

	view.CanPay = true
	view.Message = "O pagamento será concluído no ambiente seguro da InfinitePay."
	return view
}

func handleOrderCheckoutError(w http.ResponseWriter, r *http.Request, err error, page ordersdomain.ReviewPage) {
	switch {
	case errors.Is(err, ordersdomain.ErrCartRequired),
		errors.Is(err, ordersdomain.ErrEmptyCart),
		errors.Is(err, ordersdomain.ErrUnavailableItems):
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	case errors.Is(err, ordersdomain.ErrDetailsRequired):
		http.Redirect(w, r, "/checkout/dados", http.StatusSeeOther)
	case errors.Is(err, ordersdomain.ErrShippingRequired),
		errors.Is(err, ordersdomain.ErrShippingExpired),
		errors.Is(err, ordersdomain.ErrShippingChanged):
		http.Redirect(w, r, "/checkout/frete", http.StatusSeeOther)
	case errors.Is(err, ordersdomain.ErrStaleReview):
		setCheckoutPrivateCache(w)
		renderHTML(w, r, http.StatusConflict, templates.CheckoutReview(page))
	default:
		log.Print("order checkout unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutReviewUnavailable())
	}
}

func handleOrderPageError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ordersdomain.ErrInvalidOrderID),
		errors.Is(err, ordersdomain.ErrNotFound):
		http.NotFound(w, r)
	default:
		log.Print("order page unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.OrderUnavailable())
	}
}

func handleOrderTrackingPageError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ordersdomain.ErrInvalidTrackingID),
		errors.Is(err, ordersdomain.ErrNotFound):
		log.Print("order tracking not found")
		http.NotFound(w, r)
	default:
		log.Print("order tracking unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.OrderTrackingUnavailable())
	}
}
