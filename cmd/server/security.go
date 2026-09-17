package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/internal/seo"
)

const globalMaxRequestBodyBytes = 1 << 20

func securityMiddleware(next http.Handler, siteURL string, supabaseURL string) http.Handler {
	policy := contentSecurityPolicy(supabaseURL)
	canonical, hasCanonical := configuredSiteURL(siteURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		origin := ""
		if hasCanonical {
			origin = canonical.Scheme + "://" + canonical.Host
		}
		noIndex := isNoIndexPath(r.URL.Path)
		r = r.WithContext(seo.WithContext(requestIDContext(r.Context(), requestID), origin, noIndex))
		w.Header().Set("X-Request-ID", requestID)
		if noIndex {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		}
		setGlobalSecurityHeaders(w.Header(), policy)
		if hasCanonical {
			if destination, ok := canonicalHostRedirectURL(r, canonical); ok {
				http.Redirect(w, r, destination, http.StatusPermanentRedirect)
				return
			}
		}
		if !limitRequestBody(w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isNoIndexPath(path string) bool {
	for _, prefix := range []string{"/admin", "/checkout", "/carrinho", "/pedido", "/acompanhar", "/pagamento"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	return false
}

func canonicalHostRedirectURL(r *http.Request, canonical *url.URL) (string, bool) {
	if r == nil || r.URL == nil || canonical == nil || !canonicalRedirectMethod(r.Method) || sameHost(r.Host, canonical.Host) {
		return "", false
	}

	destination := url.URL{
		Scheme:   canonical.Scheme,
		Host:     canonical.Host,
		Path:     r.URL.Path,
		RawPath:  r.URL.RawPath,
		RawQuery: r.URL.RawQuery,
	}
	if destination.Path == "" {
		destination.Path = "/"
	}

	return destination.String(), true
}

func canonicalRedirectMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func setGlobalSecurityHeaders(header http.Header, contentSecurityPolicy string) {
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	header.Set("Content-Security-Policy", contentSecurityPolicy)
}

func limitRequestBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}

	if r.ContentLength > globalMaxRequestBodyBytes {
		http.Error(w, "request entity too large", http.StatusRequestEntityTooLarge)
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, globalMaxRequestBodyBytes)
	return true
}

func writeInvalidBodyError(w http.ResponseWriter, message string, err error) {
	if isMaxBytesError(err) {
		http.Error(w, message, http.StatusRequestEntityTooLarge)
		return
	}

	http.Error(w, message, http.StatusBadRequest)
}

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func contentSecurityPolicy(supabaseURL string) string {
	imgSrc := "img-src 'self' data:"
	connectSrc := "connect-src 'self'"
	if origin := configuredOrigin(supabaseURL); origin != "" {
		imgSrc += " " + origin
		connectSrc += " " + origin
	}
	formAction := "form-action 'self'"
	for _, origin := range paymentsdomain.CheckoutAllowedOrigins() {
		formAction += " " + origin
	}

	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		imgSrc,
		connectSrc,
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'none'",
		formAction,
	}, "; ")
}

func configuredOrigin(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || !webScheme(parsed.Scheme) {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}
