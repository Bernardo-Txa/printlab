package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type checkoutShippingService interface {
	Page(ctx context.Context, tokenHash []byte, selected bool) (shipping.CheckoutShippingPage, error)
	Select(ctx context.Context, tokenHash []byte, serviceCode string) (shipping.SelectResult, error)
}

func checkoutShippingPageHandler(service checkoutShippingService, cookies *cartdomain.CookieManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCheckoutPrivateCache(w)

		_, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutShippingUnavailable())
			return
		}

		page, err := service.Page(r.Context(), tokenHash, r.URL.Query().Get("selecionado") == "1")
		if err != nil {
			handleCheckoutShippingError(w, r, err, page)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.CheckoutShipping(page))
	}
}

func selectShippingHandler(service checkoutShippingService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCheckoutPrivateCache(w)

		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		_, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutShippingUnavailable())
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid shipping request", http.StatusBadRequest)
			return
		}

		result, err := service.Select(r.Context(), tokenHash, strings.TrimSpace(r.PostFormValue("service_code")))
		if err != nil {
			handleCheckoutShippingError(w, r, err, result.Page)
			return
		}

		http.Redirect(w, r, "/checkout/frete?selecionado=1", http.StatusSeeOther)
	}
}

func handleCheckoutShippingError(w http.ResponseWriter, r *http.Request, err error, page shipping.CheckoutShippingPage) {
	switch {
	case errors.Is(err, shipping.ErrCartRequired),
		errors.Is(err, shipping.ErrEmptyCart),
		errors.Is(err, shipping.ErrUnavailableItems):
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	case errors.Is(err, shipping.ErrDetailsRequired):
		http.Redirect(w, r, "/checkout/dados", http.StatusSeeOther)
	case errors.Is(err, shipping.ErrInvalidService):
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutShipping(page))
	case errors.Is(err, shipping.ErrNoQuotes):
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutShipping(page))
	default:
		log.Print("shipping unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutShippingUnavailable())
	}
}
