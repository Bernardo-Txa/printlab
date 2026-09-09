package products

import "testing"

func TestEffectivePriceCents(t *testing.T) {
	product := Product{PriceCents: 3990}
	override := int64(5990)
	zero := int64(0)

	tests := []struct {
		name    string
		variant ProductVariant
		want    int64
	}{
		{name: "fallback to product base price", variant: ProductVariant{}, want: 3990},
		{name: "variant override", variant: ProductVariant{PriceCents: &override}, want: 5990},
		{name: "zero is explicit override", variant: ProductVariant{PriceCents: &zero}, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EffectivePriceCents(product, test.variant); got != test.want {
				t.Fatalf("expected %d, got %d", test.want, got)
			}
		})
	}
}

func TestTotalFilamentWeightMg(t *testing.T) {
	filaments := []VariantFilament{
		{EstimatedWeightMg: 32000},
		{EstimatedWeightMg: 7000},
		{EstimatedWeightMg: 3000},
	}

	if got := TotalFilamentWeightMg(filaments); got != 42000 {
		t.Fatalf("expected 42000 mg, got %d", got)
	}
}

func TestFormatWeightGrams(t *testing.T) {
	tests := []struct {
		weightMg int64
		want     string
	}{
		{weightMg: 42000, want: "42 g"},
		{weightMg: 3250, want: "3,25 g"},
		{weightMg: 125500, want: "125,5 g"},
	}

	for _, test := range tests {
		if got := FormatWeightGrams(test.weightMg); got != test.want {
			t.Fatalf("expected %q, got %q", test.want, got)
		}
	}
}

func TestFormatPrintTime(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{minutes: 60, want: "1h"},
		{minutes: 275, want: "4h 35min"},
		{minutes: 45, want: "45min"},
	}

	for _, test := range tests {
		if got := FormatPrintTime(test.minutes); got != test.want {
			t.Fatalf("expected %q, got %q", test.want, got)
		}
	}
}
