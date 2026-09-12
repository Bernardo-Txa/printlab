package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

type cartService interface {
	View(ctx context.Context, tokenHash []byte) (cartdomain.CartView, error)
	CheckoutCart(ctx context.Context, tokenHash []byte) (cartdomain.Cart, cartdomain.CartView, error)
	Renew(ctx context.Context, cartID string) (cartdomain.Cart, error)
	Add(ctx context.Context, tokenHash []byte, input cartdomain.AddItemInput) (cartdomain.Cart, error)
	UpdateQuantity(ctx context.Context, tokenHash []byte, itemID string, quantity int) (*cartdomain.Cart, error)
	RemoveItem(ctx context.Context, tokenHash []byte, itemID string) (*cartdomain.Cart, error)
}

func cartPageHandler(service cartService, cookies *cartdomain.CookieManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookies = ensureCartCookies(cookies)
		token, ok := cookies.ReadToken(r)
		if !ok {
			renderHTML(w, r, http.StatusOK, templates.CartPage(cartdomain.EmptyView()))
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
			return
		}

		tokenHash, err := cookies.HashToken(token)
		if err != nil {
			renderHTML(w, r, http.StatusOK, templates.CartPage(cartdomain.EmptyView()))
			return
		}

		view, err := service.View(r.Context(), tokenHash)
		if err != nil {
			log.Print("cart unavailable")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.CartPage(view))
	}
}

func addCartItemHandler(service cartService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
			return
		}

		quantity, ok := quantityFromRequest(w, r)
		if !ok {
			return
		}

		cookies = ensureCartCookies(cookies)
		token, ok := cookies.ReadToken(r)
		if !ok {
			generatedToken, err := cookies.GenerateToken()
			if err != nil {
				log.Print("cart token generation failed")
				renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
				return
			}
			token = generatedToken
		}

		tokenHash, err := cookies.HashToken(token)
		if err != nil {
			http.Error(w, "invalid cart request", http.StatusBadRequest)
			return
		}

		activeCart, err := service.Add(r.Context(), tokenHash, cartdomain.AddItemInput{
			ProductSlug: r.PostFormValue("product_slug"),
			VariantSlug: r.PostFormValue("variant_slug"),
			Quantity:    quantity,
		})
		if err != nil {
			handleCartMutationError(w, r, err)
			return
		}

		cookies.SetCookie(w, token, activeCart.ExpiresAt)
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	}
}

func updateCartItemQuantityHandler(service cartService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		itemID := r.PathValue("id")
		if !validUUID(itemID) {
			http.NotFound(w, r)
			return
		}

		quantity, ok := quantityFromRequest(w, r)
		if !ok {
			return
		}

		cookies = ensureCartCookies(cookies)
		token, ok := cookies.ReadToken(r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
			return
		}

		tokenHash, err := cookies.HashToken(token)
		if err != nil {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		activeCart, err := service.UpdateQuantity(r.Context(), tokenHash, itemID, quantity)
		if err != nil {
			handleCartMutationError(w, r, err)
			return
		}

		if activeCart != nil {
			cookies.SetCookie(w, token, activeCart.ExpiresAt)
		}
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	}
}

func removeCartItemHandler(service cartService, cookies *cartdomain.CookieManager, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		itemID := r.PathValue("id")
		if !validUUID(itemID) {
			http.NotFound(w, r)
			return
		}

		cookies = ensureCartCookies(cookies)
		token, ok := cookies.ReadToken(r)
		if !ok {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
			return
		}

		tokenHash, err := cookies.HashToken(token)
		if err != nil {
			http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
			return
		}

		activeCart, err := service.RemoveItem(r.Context(), tokenHash, itemID)
		if err != nil {
			handleCartMutationError(w, r, err)
			return
		}

		if activeCart != nil {
			cookies.SetCookie(w, token, activeCart.ExpiresAt)
		}
		http.Redirect(w, r, "/carrinho", http.StatusSeeOther)
	}
}

func quantityFromRequest(w http.ResponseWriter, r *http.Request) (int, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid cart request", http.StatusBadRequest)
		return 0, false
	}

	quantity, err := strconv.Atoi(strings.TrimSpace(r.PostFormValue("quantity")))
	if err != nil || quantity < cartdomain.MinQuantity || quantity > cartdomain.MaxQuantity {
		http.Error(w, "invalid cart request", http.StatusBadRequest)
		return 0, false
	}

	return quantity, true
}

func handleCartMutationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, cartdomain.ErrInvalidQuantity),
		errors.Is(err, cartdomain.ErrInvalidProduct),
		errors.Is(err, cartdomain.ErrInvalidVariant),
		errors.Is(err, cartdomain.ErrProductUnavailable),
		errors.Is(err, cartdomain.ErrVariantRequired),
		errors.Is(err, cartdomain.ErrVariantUnavailable),
		errors.Is(err, cartdomain.ErrQuantityLimit):
		http.Error(w, "invalid cart request", http.StatusBadRequest)
	default:
		log.Print("cart mutation unavailable")
		renderHTML(w, r, http.StatusServiceUnavailable, templates.CartUnavailable())
	}
}

func ensureCartCookies(cookies *cartdomain.CookieManager) *cartdomain.CookieManager {
	if cookies != nil {
		return cookies
	}

	return cartdomain.NewCookieManager(cartdomain.CookieOptions{})
}

func secureCartCookies(cfg config.Config) bool {
	if isProductionEnv(cfg.AppEnv) || isProductionEnv(cfg.VercelEnv) {
		return true
	}

	siteURL, err := url.Parse(strings.TrimSpace(cfg.SiteURL))
	return err == nil && siteURL.Scheme == "https"
}

func isProductionEnv(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "production" || value == "prod"
}

func validMutationSource(r *http.Request, siteURL string) bool {
	if r == nil {
		return false
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		if strings.EqualFold(strings.TrimSpace(origin), "null") {
			referer := r.Header.Get("Referer")
			return referer != "" && allowedRequestSource(referer, r, siteURL)
		}
		return allowedRequestSource(origin, r, siteURL)
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		return allowedRequestSource(referer, r, siteURL)
	}

	return true
}

func allowedRequestSource(rawSource string, r *http.Request, siteURL string) bool {
	source, err := url.Parse(strings.TrimSpace(rawSource))
	if err != nil || source.Host == "" || !webScheme(source.Scheme) {
		return false
	}

	if sameHost(source.Host, r.Host) {
		return true
	}

	configured, err := url.Parse(strings.TrimSpace(siteURL))
	if err == nil && configured.Host != "" && sameHost(source.Host, configured.Host) {
		return true
	}

	return false
}

func webScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

func sameHost(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func validUUID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}

	return true
}
