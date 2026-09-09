package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const checkoutPrivateCacheControl = "private, no-store"

type checkoutDetailsService interface {
	Page(ctx context.Context, tokenHash []byte, saved bool) (customers.CheckoutPage, error)
	Save(ctx context.Context, tokenHash []byte, input customers.CheckoutInput) (customers.SaveResult, error)
}

func checkoutDetailsPageHandler(service checkoutDetailsService, cookies *cartdomain.CookieManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}
		setCheckoutPrivateCache(w)

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutDetailsUnavailable())
			return
		}

		page, err := service.Page(r.Context(), tokenHash, r.URL.Query().Get("salvo") == "1")
		if err != nil {
			handleCheckoutDetailsError(w, r, err, customers.CheckoutPage{})
			return
		}

		renderHTML(w, r, http.StatusOK, templates.CheckoutDetails(page))
	}
}

func saveCheckoutDetailsHandler(service checkoutDetailsService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		token, tokenHash, ok := checkoutToken(cookies, r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutDetailsUnavailable())
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid customer details", http.StatusBadRequest)
			return
		}

		result, err := service.Save(r.Context(), tokenHash, checkoutInputFromRequest(r))
		if err != nil {
			handleCheckoutDetailsError(w, r, err, result.Page)
			return
		}

		ensureCartCookies(cookies).SetCookie(w, token, result.ExpiresAt)
		http.Redirect(w, r, "/checkout/frete", http.StatusSeeOther)
	}
}

func setCheckoutPrivateCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", checkoutPrivateCacheControl)
}

func checkoutToken(cookies *cartdomain.CookieManager, r *http.Request) (string, []byte, bool) {
	cookies = ensureCartCookies(cookies)
	token, ok := cookies.ReadToken(r)
	if !ok {
		return "", nil, false
	}

	tokenHash, err := cookies.HashToken(token)
	if err != nil {
		return "", nil, false
	}

	return token, tokenHash, true
}

func checkoutInputFromRequest(r *http.Request) customers.CheckoutInput {
	return customers.CheckoutInput{
		FullName:    r.PostFormValue("full_name"),
		Email:       r.PostFormValue("email"),
		Phone:       r.PostFormValue("phone"),
		CPF:         r.PostFormValue("cpf"),
		PostalCode:  r.PostFormValue("postal_code"),
		Street:      r.PostFormValue("street"),
		Number:      r.PostFormValue("number"),
		Complement:  r.PostFormValue("complement"),
		District:    r.PostFormValue("district"),
		City:        r.PostFormValue("city"),
		State:       r.PostFormValue("state"),
		CountryCode: r.PostFormValue("country_code"),
	}
}

func handleCheckoutDetailsError(w http.ResponseWriter, r *http.Request, err error, page customers.CheckoutPage) {
	switch {
	case errors.Is(err, customers.ErrInvalidDetails):
		setCheckoutPrivateCache(w)
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutDetails(page))
	case errors.Is(err, customers.ErrCartRequired),
		errors.Is(err, customers.ErrEmptyCart),
		errors.Is(err, customers.ErrUnavailableItems):
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	default:
		log.Print("checkout details unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutDetailsUnavailable())
	}
}
