package shipping

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNoFittingBoxDiagnosticsAreSafeAndExplainRotation(t *testing.T) {
	selected := SuperFreteReturnedPackage{HeightMM: 200, WidthMM: 300, LengthMM: 400}
	// Synthetic fixture, not the unknown package dimensions from production.
	quotes := []SuperFreteQuote{
		{ServiceCode: "secret-token", ServiceName: "buyer@example.com", Package: &selected},
		{ServiceCode: "2", Package: &selected},
	}
	boxes := []ShippingBox{{ID: "private-id", Name: "Rua Um\nAuthorization: secret", Slug: "52998224725", Internal: DimensionsMM{Height: 240, Width: 120, Length: 160}, External: DimensionsMM{Height: 1000, Width: 1000, Length: 1000}}}
	logs := captureShippingLogs(t, func() { logNoFittingBoxDiagnostics(quotes, selected, boxes) })
	for _, forbidden := range []string{"private-id", "Rua Um", "Authorization", "secret", "buyer@example.com", "52998224725", "1000"} {
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
	if report.QuotesWithPackage != 2 || report.CandidateBoxes != 1 || report.SelectedMM != [3]int{200, 300, 400} || len(report.Planning) != 2 {
		t.Fatalf("report=%+v", report)
	}
	box := report.Boxes[0]
	if box.InternalSortedMM != [3]int{120, 160, 240} || box.Fits || box.DeficitMM == nil || *box.DeficitMM != [3]int{80, 140, 160} {
		t.Fatalf("box=%+v", box)
	}
	if boxes[0].Internal.Height != 240 || quotes[0].Package != &selected {
		t.Fatal("diagnostics mutated inputs")
	}
}

func TestNoFittingBoxDiagnosticBoundsAndInvalidDimensions(t *testing.T) {
	quotes := make([]SuperFreteQuote, packagingDiagnosticLimit+2)
	boxes := make([]ShippingBox, packagingDiagnosticLimit+3)
	for i := range quotes {
		quotes[i].Package = &SuperFreteReturnedPackage{HeightMM: 1, WidthMM: 2, LengthMM: 3}
	}
	logs := captureShippingLogs(t, func() { logNoFittingBoxDiagnostics(quotes, SuperFreteReturnedPackage{}, boxes) })
	var report packagingDiagnostic
	if err := json.Unmarshal([]byte(strings.SplitN(logs, " data=", 2)[1]), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Planning) != 32 || len(report.Boxes) != 32 || report.PlanningOmitted != 2 || report.BoxesOmitted != 3 || report.SelectedValid {
		t.Fatalf("report=%+v", report)
	}
	if report.Boxes[0].DeficitMM != nil || report.Boxes[0].InternalValid {
		t.Fatal("invalid dimensions must not produce deficits")
	}
}

func TestPackagingFailureLogsCandidatesWithoutFinalQuote(t *testing.T) {
	selected := &SuperFreteReturnedPackage{HeightMM: 200, WidthMM: 300, LengthMM: 400}
	calculator := &fakeShippingCalculator{responses: [][]SuperFreteQuote{{{ServiceCode: "1", Package: selected}, {ServiceCode: "2", Package: selected}}}}
	service := shippingServiceFixture(t, &fakeShippingRepository{items: shippingCartItemsFixture(), boxes: shippingBoxesFixture()}, calculator)
	logs := captureShippingLogs(t, func() {
		page, err := service.Page(context.Background(), []byte("private-cart-token"), false)
		if err != nil || !page.Unavailable {
			t.Fatalf("page unavailable=%v err=%v", page.Unavailable, err)
		}
	})
	if !strings.Contains(logs, "shipping packaging diagnostic") || !strings.Contains(logs, `"quotes_with_package":2`) {
		t.Fatal("missing diagnosis")
	}
	if len(calculator.requests) != 1 {
		t.Fatal("final quote must not be called without a fitting box")
	}
}

func TestFirstReturnedPackagePreservesExistingChoice(t *testing.T) {
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
