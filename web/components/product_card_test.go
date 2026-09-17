package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/a-h/templ"
)

func TestProductGalleryPrioritizesOnlyPrimaryImage(t *testing.T) {
	html := renderComponent(t, ProductGallery([]products.ProductImage{
		{URL: "https://example.test/primary.png"},
		{URL: "https://example.test/thumbnail.png"},
	}, "Produto"))

	if !strings.Contains(html, `src="https://example.test/primary.png" alt="Produto" loading="eager" decoding="async" fetchpriority="high"`) {
		t.Fatalf("expected eager prioritized primary image, got %s", html)
	}
	mainImage, _, found := strings.Cut(html, `<div class="product-gallery-thumbs"`)
	if !found {
		t.Fatalf("expected gallery thumbnails, got %s", html)
	}
	if strings.Contains(mainImage, `loading="lazy"`) {
		t.Fatalf("expected primary image not to be lazy, got %s", mainImage)
	}
	if !strings.Contains(html, `src="https://example.test/thumbnail.png" alt="Produto" loading="lazy" decoding="async"`) {
		t.Fatalf("expected lazy asynchronous thumbnail, got %s", html)
	}
}

func TestProductCardKeepsImageLazy(t *testing.T) {
	html := renderComponent(t, ProductCard(products.Product{
		Name:         "Produto",
		Slug:         "produto",
		PrimaryImage: &products.ProductImage{URL: "https://example.test/card.png"},
	}))

	if !strings.Contains(html, `src="https://example.test/card.png" alt="Produto" loading="lazy" decoding="async"`) {
		t.Fatalf("expected lazy asynchronous card image, got %s", html)
	}
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		t.Fatalf("expected component render to succeed, got %v", err)
	}

	return buffer.String()
}
