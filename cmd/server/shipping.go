package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type checkoutShippingService interface {
	DeliveryPage(ctx context.Context, tokenHash []byte, method string) (shipping.CheckoutShippingPage, error)
	SaveAddress(ctx context.Context, tokenHash []byte, input customers.CheckoutInput) (shipping.CheckoutShippingPage, error)
	SelectDelivery(ctx context.Context, tokenHash []byte, method, serviceCode string) (shipping.SelectResult, error)
}

func checkoutShippingPageHandler(service checkoutShippingService, cookies *cartdomain.CookieManager, profiles accountProfileService) http.HandlerFunc {
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

		page, err := service.DeliveryPage(r.Context(), tokenHash, r.URL.Query().Get("delivery_method"))
		if err != nil {
			handleCheckoutShippingError(w, r, err, page)
			return
		}
		if authProfile, ok := customerauth.ProfileFromContext(r.Context()); ok && profiles != nil && page.SelectedMethod == shipping.DeliveryMethodShipping && !page.AddressForm.Found {
			if saved, found, err := profiles.Get(r.Context(), authProfile.ID); err != nil {
				logCustomerAuthError(r.Context(), "customer_profile_load_failed", err)
			} else if found && profileHasShippingAddress(saved) {
				page.AddressForm.Values = saved.CheckoutInput(authProfile.Email)
			}
		}

		renderHTML(w, r, http.StatusOK, templates.CheckoutShipping(page))
	}
}

func saveShippingAddressHandler(service checkoutShippingService, cookies *cartdomain.CookieManager, siteURL string, profiles accountProfileService) http.HandlerFunc {
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
			writeInvalidBodyError(w, "invalid shipping address request", err)
			return
		}
		page, err := service.SaveAddress(r.Context(), tokenHash, checkoutInputFromRequest(r))
		if err != nil {
			handleCheckoutShippingError(w, r, err, page)
			return
		}
		syncShippingAddressProfile(r.Context(), profiles, page.AddressForm.Values)
		http.Redirect(w, r, "/checkout/frete?delivery_method=shipping", http.StatusSeeOther)
	}
}

func syncShippingAddressProfile(ctx context.Context, profiles accountProfileService, input customers.CheckoutInput) {
	authProfile, ok := customerauth.ProfileFromContext(ctx)
	if !ok || profiles == nil {
		return
	}
	saved, found, err := profiles.Get(ctx, authProfile.ID)
	if err != nil {
		logCustomerAuthError(ctx, "customer_profile_load_failed", err)
		return
	}
	if !found {
		saved = customerprofile.Profile{AuthUserID: authProfile.ID, FullName: authProfile.Name, CountryCode: customers.CountryCodeBR}
	}
	saved.AuthUserID = authProfile.ID
	saved.PostalCode = input.PostalCode
	saved.Street = input.Street
	saved.Number = input.Number
	saved.Complement = input.Complement
	saved.District = input.District
	saved.City = input.City
	saved.State = input.State
	if input.CountryCode != "" {
		saved.CountryCode = input.CountryCode
	} else if saved.CountryCode == "" {
		saved.CountryCode = customers.CountryCodeBR
	}
	if err := profiles.Save(ctx, saved); err != nil {
		logCustomerAuthError(ctx, "customer_profile_sync_failed", err)
	}
}

func profileHasShippingAddress(profile customerprofile.Profile) bool {
	return profile.PostalCode != "" && profile.Street != "" && profile.Number != "" && profile.District != "" && profile.City != "" && profile.State != ""
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
			writeInvalidBodyError(w, "invalid shipping request", err)
			return
		}

		method := strings.TrimSpace(r.PostFormValue("delivery_method"))
		// Older shipping forms remain accepted; new forms always send an explicit method.
		if method == "" {
			method = shipping.DeliveryMethodShipping
		}
		result, err := service.SelectDelivery(r.Context(), tokenHash, method, strings.TrimSpace(r.PostFormValue("service_code")))
		if err != nil {
			handleCheckoutShippingError(w, r, err, result.Page)
			return
		}

		http.Redirect(w, r, "/checkout/revisao", http.StatusSeeOther)
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
	case errors.Is(err, shipping.ErrInvalidDeliveryMethod):
		http.Error(w, "invalid delivery method", http.StatusBadRequest)
	case errors.Is(err, shipping.ErrAddressRequired):
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutShipping(page))
	case errors.Is(err, shipping.ErrInvalidService):
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutShipping(page))
	case errors.Is(err, shipping.ErrNoQuotes):
		renderHTML(w, r, http.StatusBadRequest, templates.CheckoutShipping(page))
	default:
		log.Print("shipping unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.CheckoutShippingUnavailable())
	}
}
