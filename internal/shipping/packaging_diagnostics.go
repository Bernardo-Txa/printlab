package shipping

import (
	"encoding/json"
	"log"
)

// Temporary incident diagnostics. Bound output and accept only typed dimensions
// and positional indexes: never include names, identifiers or request/API bodies.
const packagingDiagnosticLimit = 32

type planningDimensionsDiagnostic struct {
	QuoteIndex   int    `json:"quote_index"`
	DimensionsMM [3]int `json:"dimensions_hwl_mm"`
}

type boxFitDiagnostic struct {
	BoxIndex         int    `json:"box_index"`
	InternalMM       [3]int `json:"internal_hwl_mm"`
	InternalSortedMM [3]int `json:"internal_sorted_mm"`
	InternalValid    bool   `json:"internal_valid"`
	Fits             bool   `json:"fits"`
	// Sorted smallest/middle/largest axes, not the original H/W/L labels.
	DeficitMM *[3]int `json:"deficit_sorted_mm,omitempty"`
}

type packagingDiagnostic struct {
	SelectedMM        [3]int                         `json:"selected_hwl_mm"`
	SelectedSortedMM  [3]int                         `json:"selected_sorted_mm"`
	SelectedValid     bool                           `json:"selected_valid"`
	QuotesWithPackage int                            `json:"quotes_with_package"`
	Planning          []planningDimensionsDiagnostic `json:"planning"`
	PlanningOmitted   int                            `json:"planning_omitted"`
	CandidateBoxes    int                            `json:"candidate_boxes"`
	Boxes             []boxFitDiagnostic             `json:"boxes"`
	BoxesOmitted      int                            `json:"boxes_omitted"`
}

func logNoFittingBoxDiagnostics(quotes []SuperFreteQuote, selected SuperFreteReturnedPackage, boxes []ShippingBox) {
	dimensions := DimensionsMM{Height: selected.HeightMM, Width: selected.WidthMM, Length: selected.LengthMM}
	report := packagingDiagnostic{
		SelectedMM:       [3]int{dimensions.Height, dimensions.Width, dimensions.Length},
		SelectedSortedMM: sortedDimensions(dimensions),
		SelectedValid:    dimensions.Valid(),
		CandidateBoxes:   len(boxes),
		Planning:         make([]planningDimensionsDiagnostic, 0),
		Boxes:            make([]boxFitDiagnostic, 0),
	}
	for i, quote := range quotes {
		if quote.Package == nil {
			continue
		}
		report.QuotesWithPackage++
		if len(report.Planning) >= packagingDiagnosticLimit {
			report.PlanningOmitted++
			continue
		}
		report.Planning = append(report.Planning, planningDimensionsDiagnostic{
			QuoteIndex: i, DimensionsMM: [3]int{quote.Package.HeightMM, quote.Package.WidthMM, quote.Package.LengthMM},
		})
	}
	for i, box := range boxes {
		if len(report.Boxes) >= packagingDiagnosticLimit {
			report.BoxesOmitted++
			continue
		}
		candidate := boxFitDiagnostic{
			BoxIndex: i, InternalMM: [3]int{box.Internal.Height, box.Internal.Width, box.Internal.Length},
			InternalSortedMM: sortedDimensions(box.Internal), InternalValid: box.Internal.Valid(),
			Fits: FitsInside(dimensions, box.Internal),
		}
		if report.SelectedValid && candidate.InternalValid {
			var deficit [3]int
			for axis := range deficit {
				deficit[axis] = max(0, report.SelectedSortedMM[axis]-candidate.InternalSortedMM[axis])
			}
			candidate.DeficitMM = &deficit
		}
		report.Boxes = append(report.Boxes, candidate)
	}
	// One record keeps the selected package and candidates together under concurrency.
	encoded, err := json.Marshal(report)
	if err != nil {
		return
	}
	log.Printf("shipping packaging diagnostic reason=no_fitting_box selection=first_returned_package data=%s", encoded)
}
