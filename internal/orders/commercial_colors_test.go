package orders

import (
	"errors"
	"testing"
)

func TestReviewCommercialColorIsIndependentOfRecipe(t *testing.T) {
	raw := rawReviewItem{ProductID: "product", ProductIsActive: true, ProductPriceCents: 100, Quantity: 1, ColorID: "blue", ColorName: "Azul comercial", ColorSlug: "azul", ColorAvailable: true}
	recipe := []ReviewItemFilament{{ColorName: "Preto producao", EstimatedWeightMgPerUnit: 1000}}
	item, err := prepareReviewItem(raw, recipe, 0)
	if err != nil || item.ColorID != "blue" || item.ColorName != raw.ColorName || item.Filaments[0].ColorName != recipe[0].ColorName {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	raw.ColorAvailable = false
	if _, err := prepareReviewItem(raw, recipe, 0); !errors.Is(err, ErrUnavailableItems) {
		t.Fatalf("invalid color accepted: %v", err)
	}
	raw.ColorID, raw.ColorName, raw.ColorSlug = "", "", ""
	if _, err := prepareReviewItem(raw, recipe, 0); err != nil {
		t.Fatalf("colorless product rejected: %v", err)
	}
}

func TestReviewFingerprintIncludesCommercialColor(t *testing.T) {
	for _, field := range []string{"id", "name", "slug"} {
		t.Run(field, func(t *testing.T) {
			page := reviewPageFixtureForFingerprint()
			before, err := buildReviewFingerprint(page, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "id":
				page.Items[0].ColorID = "blue"
			case "name":
				page.Items[0].ColorName = "Azul"
			case "slug":
				page.Items[0].ColorSlug = "azul"
			}
			after, err := buildReviewFingerprint(page, nil)
			if err != nil || before == after {
				t.Fatalf("color change missing from fingerprint: %v", err)
			}
		})
	}
}
