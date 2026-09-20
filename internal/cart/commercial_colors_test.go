package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

func TestAddCommercialColorsSeparatesItemsAndMergesSameColor(t *testing.T) {
	r := newFakeRepository()
	r.products["dragao"] = ProductForAdd{ID: "product", IsActive: true, Colors: []products.ProductAvailableColor{
		{Color: products.Color{ID: "blue", Slug: "azul"}}, {Color: products.Color{ID: "red", Slug: "vermelho"}},
	}}
	s := NewService(r, WithClock(fixedClock()))
	for _, color := range []string{"azul", "vermelho", "azul", ""} {
		if _, err := s.Add(context.Background(), testHash(1), AddItemInput{ProductSlug: "dragao", ColorSlug: color, Quantity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.quantitiesByKey) != 3 {
		t.Fatalf("expected three independent lines, got %v", r.quantitiesByKey)
	}
	for _, invalid := range []string{"outra-cor", "../azul"} {
		if _, err := s.Add(context.Background(), testHash(1), AddItemInput{ProductSlug: "dragao", ColorSlug: invalid, Quantity: 1}); !errors.Is(err, ErrInvalidColor) {
			t.Fatalf("invalid color: %v", err)
		}
	}
	if r.addCalls != 4 {
		t.Fatal("invalid colors reached persistence")
	}
}

func TestCartCommercialColorAvailability(t *testing.T) {
	s := NewService(nil)
	item := StoredItem{Product: ProductSnapshot{ID: "product", IsActive: true, PriceCents: 100}, Quantity: 1, ColorID: "blue", ColorName: "Azul", ColorAvailable: true}
	line, err := s.prepareLine(item)
	if err != nil || !line.Available || line.ColorName != "Azul" {
		t.Fatalf("line=%+v err=%v", line, err)
	}
	item.ColorAvailable = false
	line, err = s.prepareLine(item)
	if err != nil || line.Available {
		t.Fatalf("unavailable color accepted: %+v %v", line, err)
	}
	item.ColorID, item.ColorName = "", ""
	line, err = s.prepareLine(item)
	if err != nil || !line.Available {
		t.Fatalf("legacy colorless line rejected: %+v %v", line, err)
	}
}
