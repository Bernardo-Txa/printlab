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
	if box.ID != "box-a" {
		t.Fatalf("expected tie to resolve by volume, sort, external volume, weight, name and id, got %q", box.ID)
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

func TestSelectSmallestBoxFitsReferenceDimensions(t *testing.T) {
	product := DimensionsMM{Height: 100, Width: 70, Length: 70}
	box := ShippingBox{ID: "reference", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}}
	selected, err := SelectSmallestBox(product, []ShippingBox{box})
	if err != nil {
		t.Fatalf("expected 183g 100x70x70 dimensions to fit 100x200x200 internal box, got %v", err)
	}
	if selected.ID != box.ID {
		t.Fatalf("expected reference box, got %#v", selected)
	}
}

func TestSelectSmallestBoxRejectsNonFittingReferenceDimensions(t *testing.T) {
	_, err := SelectSmallestBox(DimensionsMM{Height: 201, Width: 70, Length: 70}, []ShippingBox{{ID: "reference", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}}})
	if !errors.Is(err, ErrNoFittingBox) {
		t.Fatalf("expected non-fitting package to be rejected, got %v", err)
	}
}

func TestSelectShippingBoxForProductsRotatesSingleUnit(t *testing.T) {
	item := QuoteProduct{Quantity: 1, Profile: ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 100, Width: 70, Length: 70}}}
	box := ShippingBox{ID: "reference", Internal: DimensionsMM{Height: 80, Width: 250, Length: 250}}
	selected, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{box})
	if err != nil {
		t.Fatalf("expected rotated 100x70x70 item to fit 80x250x250 box, got %v", err)
	}
	if selected.ID != box.ID {
		t.Fatalf("expected reference box, got %#v", selected)
	}
}

func TestSelectShippingBoxForProductsRejectsRotatedOversizedUnit(t *testing.T) {
	item := QuoteProduct{Quantity: 1, Profile: ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 100, Width: 70, Length: 70}}}
	_, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{{ID: "too-low", Internal: DimensionsMM{Height: 60, Width: 250, Length: 250}}})
	if !errors.Is(err, ErrNoFittingBox) {
		t.Fatalf("expected box with every axis below 70mm to be rejected, got %v", err)
	}
}

func TestSelectShippingBoxForProductsChoosesSmallestCompatibleRealBox(t *testing.T) {
	item := QuoteProduct{Quantity: 1, Profile: ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 100, Width: 70, Length: 70}}}
	selected, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{
		{ID: "tall", Internal: DimensionsMM{Height: 80, Width: 250, Length: 250}, External: DimensionsMM{Height: 80, Width: 250, Length: 250}, SortOrder: 1},
		{ID: "compact", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}, External: DimensionsMM{Height: 100, Width: 200, Length: 200}, SortOrder: 1},
		{ID: "large", Internal: DimensionsMM{Height: 300, Width: 300, Length: 300}, External: DimensionsMM{Height: 300, Width: 300, Length: 300}, SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("expected compatible box, got %v", err)
	}
	if selected.ID != "compact" {
		t.Fatalf("expected smallest compatible internal volume, got %q", selected.ID)
	}
}

func TestSelectShippingBoxForProductsDoesNotMergeQuantitiesIntoOneDimension(t *testing.T) {
	item := QuoteProduct{Quantity: 2, Profile: ShippingProfile{WeightG: 100, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}}
	selected, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{{ID: "two-slots", Internal: DimensionsMM{Height: 100, Width: 100, Length: 200}}})
	if err != nil {
		t.Fatalf("expected two units to be packed as separate cuboids, got %v", err)
	}
	if selected.ID != "two-slots" {
		t.Fatalf("expected two-slot box, got %q", selected.ID)
	}
}

func TestSelectShippingBoxForProductsPacksDifferentProductsWithoutOverlap(t *testing.T) {
	items := []QuoteProduct{
		{Quantity: 1, Profile: ShippingProfile{WeightG: 100, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}},
		{Quantity: 1, Profile: ShippingProfile{WeightG: 50, Dimensions: DimensionsMM{Height: 50, Width: 100, Length: 100}}},
	}
	selected, err := SelectShippingBoxForProducts(items, []ShippingBox{{ID: "stacked", Internal: DimensionsMM{Height: 150, Width: 100, Length: 100}}})
	if err != nil {
		t.Fatalf("expected different products to fit without overlap, got %v", err)
	}
	if selected.ID != "stacked" {
		t.Fatalf("expected stacked box, got %q", selected.ID)
	}
}

func TestSelectShippingBoxForProductsRejectsItemsThatIndividuallyFitButOverlapTogether(t *testing.T) {
	items := []QuoteProduct{
		{Quantity: 1, Profile: ShippingProfile{WeightG: 100, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}},
		{Quantity: 1, Profile: ShippingProfile{WeightG: 100, Dimensions: DimensionsMM{Height: 100, Width: 100, Length: 100}}},
	}
	_, err := SelectShippingBoxForProducts(items, []ShippingBox{{ID: "single-slot", Internal: DimensionsMM{Height: 100, Width: 100, Length: 100}}})
	if !errors.Is(err, ErrNoFittingBox) {
		t.Fatalf("expected overlapping arrangement to be rejected, got %v", err)
	}
}

func TestSelectShippingBoxForProductsIsDeterministic(t *testing.T) {
	items := []QuoteProduct{{Quantity: 1, Profile: ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 100, Width: 70, Length: 70}}}}
	boxes := []ShippingBox{
		{ID: "b", Name: "B", Slug: "b", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}, External: DimensionsMM{Height: 100, Width: 200, Length: 200}, PackagingWeightG: 120, SortOrder: 2},
		{ID: "a", Name: "A", Slug: "a", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}, External: DimensionsMM{Height: 100, Width: 200, Length: 200}, PackagingWeightG: 120, SortOrder: 2},
	}
	for i := 0; i < 10; i++ {
		selected, err := SelectShippingBoxForProducts(items, boxes)
		if err != nil {
			t.Fatalf("expected deterministic fit, got %v", err)
		}
		if selected.ID != "a" {
			t.Fatalf("expected stable id tie-break, got %q", selected.ID)
		}
	}
}

func TestSelectShippingBoxForProductsHandlesLargeQuantityConservatively(t *testing.T) {
	item := QuoteProduct{Quantity: 99, Profile: ShippingProfile{WeightG: 1, Dimensions: DimensionsMM{Height: 10, Width: 10, Length: 10}}}
	selected, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{{ID: "grid", Internal: DimensionsMM{Height: 50, Width: 50, Length: 50}}})
	if err != nil {
		t.Fatalf("expected 99 small units to fit without explosive search, got %v", err)
	}
	if selected.ID != "grid" {
		t.Fatalf("expected grid box, got %q", selected.ID)
	}
}

func TestSelectShippingBoxForProductsPrintLabRealFixture(t *testing.T) {
	item := QuoteProduct{Quantity: 1, Profile: ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 100, Width: 70, Length: 70}}}
	selected, err := SelectShippingBoxForProducts([]QuoteProduct{item}, []ShippingBox{
		{ID: "80x250x250", Internal: DimensionsMM{Height: 80, Width: 250, Length: 250}, External: DimensionsMM{Height: 80, Width: 250, Length: 250}},
		{ID: "100x200x200", Internal: DimensionsMM{Height: 100, Width: 200, Length: 200}, External: DimensionsMM{Height: 100, Width: 200, Length: 200}},
		{ID: "150x200x20", Internal: DimensionsMM{Height: 150, Width: 200, Length: 20}, External: DimensionsMM{Height: 150, Width: 200, Length: 20}},
	})
	if err != nil {
		t.Fatalf("expected PrintLab real fixture to find a valid box, got %v", err)
	}
	if selected.ID != "100x200x200" {
		t.Fatalf("expected smallest compatible real box, got %q", selected.ID)
	}
}

func TestEffectiveShippingProfileFallback(t *testing.T) {
	productProfile := ShippingProfile{WeightG: 280, Dimensions: DimensionsMM{Height: 90, Width: 105, Length: 210}}

	profile, ok := EffectiveShippingProfile(CartItem{ProductProfile: &productProfile})
	if !ok || profile.WeightG != productProfile.WeightG {
		t.Fatalf("expected product profile fallback, got %#v ok=%v", profile, ok)
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

func TestDeliveryTimeLabelUsesPtBRAccents(t *testing.T) {
	one := 1
	two := 2
	if got := DeliveryTimeLabel(&one); got != "1 dia útil" {
		t.Fatalf("expected singular accented delivery time, got %q", got)
	}
	if got := DeliveryTimeLabel(&two); got != "2 dias úteis" {
		t.Fatalf("expected plural accented delivery time, got %q", got)
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
	changedBox.Packaging.ID = "box-b"
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
			ProductProfile: &ShippingProfile{
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

func TestFallbackPackageForProductsUsesConservativeVilaNatalinaExample(t *testing.T) {
	pack, err := FallbackPackageForProducts([]QuoteProduct{{
		Quantity: 1,
		Profile:  ShippingProfile{WeightG: 264, Dimensions: DimensionsMM{Height: 135, Width: 298, Length: 334}},
	}})
	if err != nil {
		t.Fatalf("expected fallback package, got %v", err)
	}
	if pack.PackagingSource != PackagingSourceFallback || pack.Box.ID != "" {
		t.Fatalf("expected fallback source without real box id, got %#v", pack)
	}
	if pack.Dimensions != (DimensionsMM{Height: 180, Width: 340, Length: 380}) {
		t.Fatalf("expected rounded fallback dimensions, got %#v", pack.Dimensions)
	}
	if pack.PackagingWeightG != 150 || pack.WeightG != 450 {
		t.Fatalf("expected fallback weights 150/450g, got packaging=%d total=%d", pack.PackagingWeightG, pack.WeightG)
	}
}

func TestFallbackPackageForProductsSumsUnitLength(t *testing.T) {
	pack, err := FallbackPackageForProducts([]QuoteProduct{{
		Quantity: 2,
		Profile:  ShippingProfile{WeightG: 100, Dimensions: DimensionsMM{Height: 50, Width: 80, Length: 120}},
	}})
	if err != nil {
		t.Fatalf("expected fallback package, got %v", err)
	}
	if pack.Dimensions != (DimensionsMM{Height: 90, Width: 120, Length: 300}) {
		t.Fatalf("expected two unit lengths plus external margin rounded up, got %#v", pack.Dimensions)
	}
}

func TestFallbackPackageForProductsHandlesMultipleProducts(t *testing.T) {
	pack, err := FallbackPackageForProducts([]QuoteProduct{
		{Quantity: 1, Profile: ShippingProfile{WeightG: 80, Dimensions: DimensionsMM{Height: 30, Width: 60, Length: 90}}},
		{Quantity: 1, Profile: ShippingProfile{WeightG: 90, Dimensions: DimensionsMM{Height: 40, Width: 70, Length: 150}}},
	})
	if err != nil {
		t.Fatalf("expected fallback package, got %v", err)
	}
	if pack.Dimensions != (DimensionsMM{Height: 80, Width: 110, Length: 300}) {
		t.Fatalf("expected multiple products to share max height/width and summed length, got %#v", pack.Dimensions)
	}
	if pack.WeightG != 350 {
		t.Fatalf("expected total fallback weight rounded to 350g, got %d", pack.WeightG)
	}
}

func TestFallbackPackageForProductsRoundsDimensionsUp(t *testing.T) {
	pack, err := FallbackPackageForProducts([]QuoteProduct{{
		Quantity: 1,
		Profile:  ShippingProfile{WeightG: 10, Dimensions: DimensionsMM{Height: 101, Width: 142, Length: 213}},
	}})
	if err != nil {
		t.Fatalf("expected fallback package, got %v", err)
	}
	if pack.Dimensions != (DimensionsMM{Height: 150, Width: 190, Length: 260}) {
		t.Fatalf("expected dimensions rounded to next 10mm, got %#v", pack.Dimensions)
	}
}

func TestFallbackPackagingWeightRules(t *testing.T) {
	tests := []struct {
		name           string
		goodsWeightG   int64
		units          int
		wantPackagingG int64
	}{
		{name: "minimum 150g", goodsWeightG: 264, units: 1, wantPackagingG: 150},
		{name: "twenty five percent wins", goodsWeightG: 1000, units: 1, wantPackagingG: 250},
		{name: "fifty grams per unit wins", goodsWeightG: 100, units: 5, wantPackagingG: 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackPackagingWeightG(tt.goodsWeightG, tt.units); got != tt.wantPackagingG {
				t.Fatalf("expected %dg packaging, got %dg", tt.wantPackagingG, got)
			}
		})
	}
}

func TestQuoteFingerprintDiffersForRealBoxAndFallbackPackage(t *testing.T) {
	realBox := quoteFingerprintFixture()
	fallback := quoteFingerprintFixture()
	fallback.Packaging = QuotePackageFingerprint{
		Source:           PackagingSourceFallback,
		ExternalHeightMM: 180,
		ExternalWidthMM:  340,
		ExternalLengthMM: 380,
		PackagingWeightG: 150,
	}
	first, err := BuildInputHash(realBox)
	if err != nil {
		t.Fatalf("expected real box hash, got %v", err)
	}
	assertDifferentHash(t, first, fallback)
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
		Packaging: QuotePackageFingerprint{
			Source:           PackagingSourceRealBox,
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

func TestSelectSmallestBoxDimensionalBoundaries(t *testing.T) {
	valid := DimensionsMM{Height: 100, Width: 150, Length: 200}
	tests := []struct {
		name       string
		dimensions DimensionsMM
		boxes      []ShippingBox
		wantID     string
		wantErr    error
	}{
		{name: "exact fit", dimensions: valid, boxes: []ShippingBox{{ID: "exact", Internal: valid}}, wantID: "exact"},
		{name: "rotated exact fit", dimensions: valid, boxes: []ShippingBox{{ID: "rotated", Internal: DimensionsMM{Height: 200, Width: 100, Length: 150}}}, wantID: "rotated"},
		{name: "one millimeter too large", dimensions: DimensionsMM{Height: 101, Width: 150, Length: 200}, boxes: []ShippingBox{{Internal: valid}}, wantErr: ErrNoFittingBox},
		{name: "external dimensions cannot authorize fit", dimensions: valid, boxes: []ShippingBox{{Internal: DimensionsMM{Height: 99, Width: 150, Length: 200}, External: valid}}, wantErr: ErrNoFittingBox},
		{name: "empty candidates", dimensions: valid, wantErr: ErrNoFittingBox},
		{name: "smallest fitting internal volume", dimensions: valid, boxes: []ShippingBox{
			{ID: "too-short", Internal: DimensionsMM{Height: 100, Width: 150, Length: 199}},
			{ID: "large", Internal: DimensionsMM{Height: 300, Width: 300, Length: 300}},
			{ID: "small", Internal: DimensionsMM{Height: 201, Width: 100, Length: 150}},
		}, wantID: "small"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectSmallestBox(tt.dimensions, tt.boxes)
			if !errors.Is(err, tt.wantErr) || got.ID != tt.wantID {
				t.Fatalf("box=%+v err=%v", got, err)
			}
		})
	}
	for _, invalid := range []DimensionsMM{{}, {Height: -1, Width: 150, Length: 200}, {Height: 100, Width: 0, Length: 200}, {Height: 100, Width: -1, Length: 200}, {Height: 100, Width: 150, Length: 0}, {Height: 100, Width: 150, Length: -1}} {
		if _, err := SelectSmallestBox(invalid, []ShippingBox{{Internal: valid}}); !errors.Is(err, ErrInvalidPackage) {
			t.Fatalf("invalid package %+v: %v", invalid, err)
		}
		if FitsInside(valid, invalid) || FitsInside(invalid, valid) {
			t.Fatalf("invalid dimensions fit: %+v", invalid)
		}
		if _, err := SelectSmallestBox(valid, []ShippingBox{{Internal: invalid}}); !errors.Is(err, ErrNoFittingBox) {
			t.Fatalf("invalid box %+v: %v", invalid, err)
		}
	}
}
