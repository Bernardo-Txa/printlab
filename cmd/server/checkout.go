package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customerprofile"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const checkoutPrivateCacheControl = "private, no-store"

type checkoutDetailsService interface {
	Page(ctx context.Context, tokenHash []byte, saved bool) (customers.CheckoutPage, error)
	Save(ctx context.Context, tokenHash []byte, input customers.CheckoutInput) (customers.SaveResult, error)
}

func checkoutDetailsPageHandler(service checkoutDetailsService, cookies *cartdomain.CookieManager, profiles accountProfileService, ordersService accountOrdersService) http.HandlerFunc {
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
		if profile, ok := customerauth.ProfileFromContext(r.Context()); ok && !page.Form.Found {
			page.Form.Values = checkoutFallbackInput(r.Context(), profile, profiles, ordersService)
		}

		renderHTML(w, r, http.StatusOK, templates.CheckoutDetails(page))
	}
}

func saveCheckoutDetailsHandler(service checkoutDetailsService, cookies *cartdomain.CookieManager, siteURL string, profiles accountProfileService) http.HandlerFunc {
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
			writeInvalidBodyError(w, "invalid customer details", err)
			return
		}

		result, err := service.Save(r.Context(), tokenHash, checkoutInputFromRequest(r))
		if err != nil {
			handleCheckoutDetailsError(w, r, err, result.Page)
			return
		}

		ensureCartCookies(cookies).SetCookie(w, token, result.ExpiresAt)
		if profile, ok := customerauth.ProfileFromContext(r.Context()); ok && profiles != nil {
			saved, found, err := profiles.Get(r.Context(), profile.ID)
			if err != nil {
				logCustomerAuthError(r.Context(), "customer_profile_load_failed", err)
			} else {
				if !found {
					saved = customerprofile.Profile{AuthUserID: profile.ID, CountryCode: customers.CountryCodeBR}
				}
				saved.AuthUserID = profile.ID
				saved.FullName = result.Page.Form.Values.FullName
				saved.Phone = result.Page.Form.Values.Phone
				saved.CPF = result.Page.Form.Values.CPF
				if saved.CountryCode == "" {
					saved.CountryCode = customers.CountryCodeBR
				}
				if err := profiles.Save(r.Context(), saved); err != nil {
					logCustomerAuthError(r.Context(), "customer_profile_sync_failed", err)
				}
			}
		}
		http.Redirect(w, r, "/checkout/frete", http.StatusSeeOther)
	}
}

func checkoutFallbackInput(ctx context.Context, profile customerauth.Profile, profiles accountProfileService, ordersService accountOrdersService) customers.CheckoutInput {
	if profiles != nil {
		saved, found, err := profiles.Get(ctx, profile.ID)
		if err != nil {
			logCustomerAuthError(ctx, "customer_profile_load_failed", err)
		} else if found {
			return saved.CheckoutInput(profile.Email)
		}
	}
	if snapshot, found := latestCustomerSnapshotForProfile(ctx, ordersService, profile.ID); found {
		return customerprofile.Profile{
			AuthUserID:  profile.ID,
			FullName:    snapshot.FullName,
			Phone:       snapshot.Phone,
			CPF:         snapshot.CPF,
			PostalCode:  snapshot.PostalCode,
			Street:      snapshot.Street,
			Number:      snapshot.Number,
			Complement:  snapshot.Complement,
			District:    snapshot.District,
			City:        snapshot.City,
			State:       snapshot.State,
			CountryCode: snapshot.CountryCode,
		}.CheckoutInput(profile.Email)
	}
	return customers.CheckoutInput{FullName: profile.Name, Email: profile.Email, CountryCode: customers.CountryCodeBR}
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
