package main

import (
	"encoding/xml"
	"net/http"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

func robotsHandler(siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, line := range []string{
			"User-agent: *",
			"Allow: /",
			"Disallow: /admin/",
			"Disallow: /checkout/",
			"Disallow: /carrinho",
			"Disallow: /pedido/",
			"Disallow: /acompanhar/",
			"Disallow: /pagamento/",
		} {
			_, _ = w.Write([]byte(line + "\n"))
		}
		if origin := canonicalOrigin(siteURL); origin != "" {
			_, _ = w.Write([]byte("Sitemap: " + origin + "/sitemap.xml\n"))
		}
	}
}

func sitemapHandler(service catalogService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := canonicalOrigin(siteURL)
		if origin == "" || service == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		catalog, err := service.Catalog(r.Context(), products.ListFilter{})
		if err != nil {
			logOperationalEvent(r.Context(), operationalLogLevelError, "sitemap_unavailable", "reason=catalog_unavailable")
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(xml.Header))
		_ = xml.NewEncoder(w).Encode(sitemap{
			XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
			URLs:  sitemapURLs(origin, catalog.Products),
		})
	}
}

func canonicalOrigin(siteURL string) string {
	configured, ok := configuredSiteURL(siteURL)
	if !ok {
		return ""
	}

	return configured.Scheme + "://" + configured.Host
}

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}

func sitemapURLs(origin string, productList []products.Product) []sitemapURL {
	urls := []sitemapURL{
		{Location: origin + "/"},
		{Location: origin + "/produtos"},
	}
	for _, product := range productList {
		if products.ValidSlug(product.Slug) {
			urls = append(urls, sitemapURL{Location: origin + "/produtos/" + product.Slug})
		}
	}

	return urls
}
