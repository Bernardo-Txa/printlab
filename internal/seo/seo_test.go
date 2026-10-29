package seo

import (
	"context"
	"testing"
)

func TestWithRequestPathPreservesPageMetadata(t *testing.T) {
	ctx := WithContext(context.Background(), "https://printlab.test", true)
	ctx = WithRequestPath(ctx, "/produtos/item")

	if got := RequestPath(ctx); got != "/produtos/item" {
		t.Fatalf("expected request path to be preserved, got %q", got)
	}
	if got := CanonicalURL(ctx, "/produtos/item"); got != "https://printlab.test/produtos/item" {
		t.Fatalf("expected canonical metadata to be preserved, got %q", got)
	}
	if !NoIndex(ctx) {
		t.Fatal("expected noindex metadata to be preserved")
	}
}
