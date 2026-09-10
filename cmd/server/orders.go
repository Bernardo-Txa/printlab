package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type orderReviewService interface {
	Review(ctx context.Context, tokenHash []byte, stale bool) (ordersdomain.ReviewPage, error)
	Confirm(ctx context.Context, tokenHash []byte, expectedFingerprint string) (ordersdomain.ConfirmResult, error)
	Get(ctx context.Context, orderID string) (ordersdomain.OrderPage, error)
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
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutReviewUnavailable())
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid order request", http.StatusBadRequest)
			return
		}

		result, err := service.Confirm(r.Context(), tokenHash, strings.TrimSpace(r.PostFormValue("review_fingerprint")))
		if err != nil {
			handleOrderCheckoutError(w, r, err, result.Page)
			return
		}

		if result.ExpireCookie {
			ensureCartCookies(cookies).ExpireCookie(w)
		}
		http.Redirect(w, r, "/pedido/"+result.OrderID, http.StatusSeeOther)
	}
}

func orderPageHandler(service orderReviewService) http.HandlerFunc {
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

		renderHTML(w, r, http.StatusOK, templates.OrderConfirmation(page))
	}
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
