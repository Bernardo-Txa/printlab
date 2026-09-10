package shipping

import (
	"errors"
	"math"
	"testing"
)

func TestFitsInsideAllowsRotation(t *testing.T) {
	packageDimensions := DimensionsMM{Height: 100, Width: 150, Length: 200}
	boxDimensions := DimensionsMM{Height: 200, Width: 100, Length: 150}

	if !FitsInside(packageDimensions, boxDimensions) {
		t.Fatal("expected package to fit by rotation")
	}
}

func TestFitsInsideRejectsOversizedAxis(t *testing.T) {
	if FitsInside(DimensionsMM{Height: 101, Width: 150, Length: 200}, DimensionsMM{Height: 200, Width: 100, Length: 150}) {
		t.Fatal("expected package with oversized axis not to fit")
	}
}

func TestFitsInsideRejectsVolumeOnlyFit(t *testing.T) {
	packageDimensions := DimensionsMM{Height: 10, Width: 10, Length: 1000}
	boxDimensions := DimensionsMM{Height: 100, Width: 100, Length: 100}

	if FitsInside(packageDimensions, boxDimensions) {
		t.Fatal("expected volume-only fit to be rejected")
	}
}

func TestSelectSmallestBox(t *testing.T) {
	box, err := SelectSmallestBox(DimensionsMM{Height: 100, Width: 100, Length: 100}, []ShippingBox{
		{ID: "large", Name: "Large", Internal: DimensionsMM{Height: 300, Width: 300, Length: 300}, PackagingWeightG: 100, SortOrder: 1},
		{ID: "small", Name: "Small", Internal: DimensionsMM{Height: 120, Width: 120, Length: 120}, PackagingWeightG: 200, SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("expected fitting box, got %v", err)
	}
	if box.ID != "small" {
		t.Fatalf("expected smallest internal volume, got %q", box.ID)
	}
}

func TestSelectSmallestBoxDeterministicTies(t *testing.T) {
	box, err := SelectSmallestBox(DimensionsMM{Height: 90, Width: 90, Length: 90}, []ShippingBox{
		{ID: "box-d", Name: "Beta", Internal: DimensionsMM{Height: 100, Width: 100, Length: 100}, PackagingWeightG: 90, SortOrder: 3},
		{ID: "box-c", Name: "Alpha", Internal: DimensionsMM{Height: 100, Width: 100, Length: 100}, PackagingWeightG: 90, SortOrder: 3},
		{ID: "box-b", Name: "Alpha", Internal: DimensionsMM{Height: 100, Width: 100, Length: 100}, PackagingWeightG: 90, SortOrder: 3},
		{ID: "box-a", Name: "Alpha", Internal: DimensionsMM{Height: 100, Width: 100, Length: 100}, PackagingWeightG: 120, SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("expected fitting box, got %v", err)
	}
	if box.ID != "box-b" {
		t.Fatalf("expected tie to resolve by volume, weight, sort, name and id, got %q", box.ID)
	}
}

func TestSelectSmallestBoxNoFittingBox(t *testing.T) {
	_, err := SelectSmallestBox(DimensionsMM{Height: 300, Width: 100, Length: 100}, []ShippingBox{
		{ID: "small", Internal: DimensionsMM{Height: 200, Width: 90, Length: 90}, PackagingWeightG: 100},
	})
	if !errors.Is(err, ErrNoFittingBox) {
		t.Fatalf("expected ErrNoFittingBox, got %v", err)
	}
}

func TestEffectiveShippingProfileFallback(t *testing.T) {
	productProfile := ShippingProfile{WeightG: 280, Dimensions: DimensionsMM{Height: 90, Width: 105, Length: 210}}
	variantProfile := ShippingProfile{WeightG: 300, Dimensions: DimensionsMM{Height: 100, Width: 120, Length: 220}}

	profile, ok := EffectiveShippingProfile(CartItem{ProductProfile: &productProfile})
	if !ok || profile.WeightG != productProfile.WeightG {
		t.Fatalf("expected product profile fallback, got %#v ok=%v", profile, ok)
	}

	profile, ok = EffectiveShippingProfile(CartItem{ProductProfile: &productProfile, VariantProfile: &variantProfile})
	if !ok || profile.WeightG != variantProfile.WeightG {
		t.Fatalf("expected variant profile override, got %#v ok=%v", profile, ok)
	}

	if _, ok := EffectiveShippingProfile(CartItem{}); ok {
		t.Fatal("expected missing profiles to be unavailable")
	}
}

func TestConversions(t *testing.T) {
	if GramsToKilograms(1000) != 1 {
		t.Fatal("expected 1000 g to be 1 kg")
	}
	if GramsToKilograms(285) != 0.285 {
		t.Fatal("expected 285 g to be 0.285 kg")
	}
	if MillimetersToCentimeters(100) != 10 {
		t.Fatal("expected 100 mm to be 10 cm")
	}
	if MillimetersToCentimeters(105) != 10.5 {
		t.Fatal("expected 105 mm to be 10.5 cm")
	}

	mm, err := CentimetersToMillimetersCeil("10")
	if err != nil || mm != 100 {
		t.Fatalf("expected 10 cm to be 100 mm, got %d err=%v", mm, err)
	}
	mm, err = CentimetersToMillimetersCeil("10.01")
	if err != nil || mm != 101 {
		t.Fatalf("expected 10.01 cm to be 101 mm, got %d err=%v", mm, err)
	}
}

func TestTotalPackageWeight(t *testing.T) {
	total, err := TotalPackageWeightG([]QuoteProduct{
		{Quantity: 1, Profile: ShippingProfile{WeightG: 280, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}},
		{Quantity: 2, Profile: ShippingProfile{WeightG: 35, Dimensions: DimensionsMM{Height: 50, Width: 50, Length: 50}}},
	}, 120)
	if err != nil {
		t.Fatalf("expected total weight, got %v", err)
	}
	if total != 470 {
		t.Fatalf("expected total weight 470 g, got %d", total)
	}
}

func TestTotalPackageWeightOverflow(t *testing.T) {
	_, err := TotalPackageWeightG([]QuoteProduct{
		{Quantity: 2, Profile: ShippingProfile{WeightG: math.MaxInt64/2 + 1, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}},
	}, 1)
	if !errors.Is(err, ErrAmountOverflow) {
		t.Fatalf("expected ErrAmountOverflow, got %v", err)
	}
}

func TestDecimalToCents(t *testing.T) {
	tests := []struct {
		value string
		want  int64
	}{
		{value: "18.90", want: 1890},
		{value: "18.9", want: 1890},
		{value: "0", want: 0},
		{value: "1.235", want: 124},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := DecimalToCents(tt.value)
			if err != nil {
				t.Fatalf("expected cents, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %d cents, got %d", tt.want, got)
			}
		})
	}
}

func TestBuildInputHash(t *testing.T) {
	fingerprint := quoteFingerprintFixture()
	reordered := quoteFingerprintFixture()
	reordered.Products[0], reordered.Products[1] = reordered.Products[1], reordered.Products[0]
	reordered.Services[0], reordered.Services[1] = reordered.Services[1], reordered.Services[0]

	first, err := BuildInputHash(fingerprint)
	if err != nil {
		t.Fatalf("expected hash, got %v", err)
	}
	second, err := BuildInputHash(reordered)
	if err != nil {
		t.Fatalf("expected hash, got %v", err)
	}
	if InputHashHex(first) != InputHashHex(second) {
		t.Fatal("expected deterministic hash independent from accidental ordering")
	}
	if fingerprint.Products[0].ID != "item-a" || fingerprint.Services[0] != "1" {
		t.Fatal("expected BuildInputHash not to mutate input slices")
	}

	changedQuantity := quoteFingerprintFixture()
	changedQuantity.Products[0].Quantity++
	assertDifferentHash(t, first, changedQuantity)

	changedPostalCode := quoteFingerprintFixture()
	changedPostalCode.DestinationPostalCode = "29100000"
	assertDifferentHash(t, first, changedPostalCode)

	changedVariant := quoteFingerprintFixture()
	changedVariant.Products[0].VariantID = "variant-b"
	assertDifferentHash(t, first, changedVariant)

	changedBox := quoteFingerprintFixture()
	changedBox.Box.ID = "box-b"
	assertDifferentHash(t, first, changedBox)
}

func TestBuildCartInputHashMatchesSelectionFingerprint(t *testing.T) {
	box := ShippingBox{
		ID: "box-a",
		External: DimensionsMM{
			Height: 120,
			Width:  150,
			Length: 240,
		},
		PackagingWeightG: 120,
	}
	items := []CartItem{
		{
			ID:        "item-a",
			ProductID: "product-a",
			VariantID: "variant-a",
			Quantity:  1,
			VariantProfile: &ShippingProfile{
				WeightG: 280,
				Dimensions: DimensionsMM{
					Height: 100,
					Width:  120,
					Length: 200,
				},
			},
		},
		{
			ID:        "item-b",
			ProductID: "product-b",
			Quantity:  2,
			ProductProfile: &ShippingProfile{
				WeightG: 35,
				Dimensions: DimensionsMM{
					Height: 50,
					Width:  60,
					Length: 70,
				},
			},
		},
	}

	got, err := BuildCartInputHash("01153000", "20020050", []string{"1", "2"}, items, box)
	if err != nil {
		t.Fatalf("expected cart input hash, got %v", err)
	}
	want, err := BuildInputHash(quoteFingerprintFixture())
	if err != nil {
		t.Fatalf("expected fixture hash, got %v", err)
	}
	if InputHashHex(got) != InputHashHex(want) {
		t.Fatal("expected cart input hash to match persisted selection fingerprint")
	}
}

func quoteFingerprintFixture() QuoteFingerprint {
	return QuoteFingerprint{
		OriginPostalCode:      "01153000",
		DestinationPostalCode: "20020050",
		Services:              []string{"1", "2"},
		Options: QuoteOptions{
			OwnHand:           false,
			Receipt:           false,
			UseInsuranceValue: false,
		},
		Products: []QuoteProductFingerprint{
			{ID: "item-a", ProductID: "product-a", VariantID: "variant-a", Quantity: 1, WeightG: 280, HeightMM: 100, WidthMM: 120, LengthMM: 200},
			{ID: "item-b", ProductID: "product-b", Quantity: 2, WeightG: 35, HeightMM: 50, WidthMM: 60, LengthMM: 70},
		},
		Box: QuoteBoxFingerprint{
			ID:               "box-a",
			ExternalHeightMM: 120,
			ExternalWidthMM:  150,
			ExternalLengthMM: 240,
			PackagingWeightG: 120,
		},
	}
}

func assertDifferentHash(t *testing.T, original []byte, fingerprint QuoteFingerprint) {
	t.Helper()

	changed, err := BuildInputHash(fingerprint)
	if err != nil {
		t.Fatalf("expected changed hash, got %v", err)
	}
	if InputHashHex(original) == InputHashHex(changed) {
		t.Fatal("expected changed fingerprint to produce different hash")
	}
}
