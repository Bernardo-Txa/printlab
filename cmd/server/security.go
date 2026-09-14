package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const globalMaxRequestBodyBytes = 1 << 20

func securityMiddleware(next http.Handler, supabaseURL string) http.Handler {
	policy := contentSecurityPolicy(supabaseURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setGlobalSecurityHeaders(w.Header(), policy)
		if !limitRequestBody(w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
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

	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		imgSrc,
		connectSrc,
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'none'",
		"form-action 'self'",
	}, "; ")
}

func configuredOrigin(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || !webScheme(parsed.Scheme) {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}
