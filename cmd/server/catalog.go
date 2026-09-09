package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/Bernardo-Txa/printlab/web/templates"
	"github.com/a-h/templ"
)

type catalogService interface {
	Catalog(ctx context.Context, filter products.ListFilter) (products.Catalog, error)
	Product(ctx context.Context, slug string) (products.Product, error)
}

func catalogHandler(service catalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CatalogUnavailable())
			return
		}

		catalog, err := service.Catalog(r.Context(), products.ListFilter{
			CategorySlug: r.URL.Query().Get("categoria"),
		})
		if err != nil {
			if errors.Is(err, products.ErrInvalidSlug) {
				renderHTML(w, r, http.StatusNotFound, templates.ProductNotFound())
				return
			}

			log.Print("catalog unavailable")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CatalogUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.Catalog(catalog))
	}
}

func productHandler(service catalogService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if !products.ValidSlug(slug) {
			renderHTML(w, r, http.StatusNotFound, templates.ProductNotFound())
			return
		}

		if service == nil {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CatalogUnavailable())
			return
		}

		product, err := service.Product(r.Context(), slug)
		if err != nil {
			if errors.Is(err, products.ErrInvalidSlug) || errors.Is(err, products.ErrNotFound) {
				renderHTML(w, r, http.StatusNotFound, templates.ProductNotFound())
				return
			}

			log.Print("catalog unavailable")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.CatalogUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.ProductDetail(product, productDescription(product)))
	}
}

func renderHTML(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := component.Render(r.Context(), w); err != nil {
		log.Printf("render response: %v", err)
	}
}

func productDescription(product products.Product) string {
	if product.ShortDescription != "" {
		return product.ShortDescription
	}

	return "Produto da PrintLab em impressao 3D."
}
