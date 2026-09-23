package shipping

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNoFittingBoxDiagnosticsAreSafeAndExplainRealPacking(t *testing.T) {
	items := []QuoteProduct{{
		ID:        "private-item-id",
		ProductID: "private-product-id",
		Quantity:  1,
		Profile: ShippingProfile{
			WeightG:    183,
			Dimensions: DimensionsMM{Height: 200, Width: 300, Length: 400},
		},
	}}
	boxes := []ShippingBox{{ID: "private-id", Name: "Rua Um\nAuthorization: secret", Slug: "52998224725", Internal: DimensionsMM{Height: 240, Width: 120, Length: 160}, External: DimensionsMM{Height: 1000, Width: 1000, Length: 1000}}}
	logs := captureShippingLogs(t, func() { logNoFittingBoxDiagnostics(items, boxes) })
	for _, forbidden := range []string{"private-id", "private-item-id", "private-product-id", "Rua Um", "Authorization", "secret", "52998224725", "1000"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("unexpected field in diagnostic: %q", forbidden)
		}
	}
	var report packagingDiagnostic
	data := strings.SplitN(logs, " data=", 2)
	if len(data) != 2 {
		t.Fatal("missing diagnostic")
	}
	if err := json.Unmarshal([]byte(data[1]), &report); err != nil {
		t.Fatal(err)
	}
	if report.ProductLines != 1 || report.Units != 1 || report.CandidateBoxes != 1 || report.PackingAlgorithm != "deterministic_extreme_points" || len(report.Products) != 1 {
		t.Fatalf("report=%+v", report)
	}
	if report.Products[0].DimensionsMM != [3]int{200, 300, 400} || report.Products[0].WeightG != 183 || report.Products[0].Quantity != 1 {
		t.Fatalf("product=%+v", report.Products[0])
	}
	box := report.Boxes[0]
	if box.InternalSortedMM != [3]int{120, 160, 240} || box.Fits || !box.InternalValid {
		t.Fatalf("box=%+v", box)
	}
	if boxes[0].Internal.Height != 240 || items[0].Profile.WeightG != 183 {
		t.Fatal("diagnostics mutated inputs")
	}
}

func TestNoFittingBoxDiagnosticBoundsAndInvalidDimensions(t *testing.T) {
	items := make([]QuoteProduct, packagingDiagnosticLimit+2)
	boxes := make([]ShippingBox, packagingDiagnosticLimit+3)
	for i := range items {
		items[i] = QuoteProduct{Quantity: 1, Profile: ShippingProfile{WeightG: 1, Dimensions: DimensionsMM{Height: 1, Width: 2, Length: 3}}}
	}
	logs := captureShippingLogs(t, func() { logNoFittingBoxDiagnostics(items, boxes) })
	var report packagingDiagnostic
	if err := json.Unmarshal([]byte(strings.SplitN(logs, " data=", 2)[1]), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Products) != 32 || len(report.Boxes) != 32 || report.ProductsOmitted != 2 || report.BoxesOmitted != 3 || report.Units != packagingDiagnosticLimit+2 {
		t.Fatalf("report=%+v", report)
	}
	if report.Boxes[0].InternalValid || report.Boxes[0].Fits {
		t.Fatal("invalid dimensions must not fit")
	}
}

func TestNoFittingBoxLogsFallbackAndStillUsesFinalPackageQuote(t *testing.T) {
	calculator := &fakeShippingCalculator{responses: [][]SuperFreteQuote{{}}}
	service := shippingServiceFixture(t, &fakeShippingRepository{
		items: []CartItem{{Quantity: 1, ProductProfile: &ShippingProfile{WeightG: 183, Dimensions: DimensionsMM{Height: 200, Width: 300, Length: 400}}}},
		boxes: shippingBoxesFixture(),
	}, calculator)
	logs := captureShippingLogs(t, func() {
		page, err := service.Page(context.Background(), []byte("private-cart-token"), false)
		if err != nil || !page.Unavailable {
			t.Fatalf("page unavailable=%v err=%v", page.Unavailable, err)
		}
	})
	for _, want := range []string{
		"shipping packaging fallback reason=no_fitting_box",
		"shipping packaging selected source=fallback",
		"shipping quote request stage=final",
		"stage=final reason=final_no_valid_quotes",
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected log %q, got %q", want, logs)
		}
	}
	if len(calculator.requests) != 1 || calculator.requests[0].Package == nil || len(calculator.requests[0].Products) != 0 {
		t.Fatalf("expected one package-only SuperFrete request, got %#v", calculator.requests)
	}
}

func TestFirstReturnedPackagePreservesProductsClientCompatibility(t *testing.T) {
	first := SuperFreteReturnedPackage{HeightMM: 100, WidthMM: 150, LengthMM: 200}
	other := SuperFreteReturnedPackage{HeightMM: 200, WidthMM: 300, LengthMM: 400}
	quotes := []SuperFreteQuote{{}, {Package: &first}, {Package: &other}}
	got, ok := firstReturnedPackage(quotes)
	if !ok || !reflect.DeepEqual(got, first) {
		t.Fatalf("choice changed: %+v %v", got, ok)
	}
	if _, ok := firstReturnedPackage([]SuperFreteQuote{{}}); ok {
		t.Fatal("missing package accepted")
	}
}
