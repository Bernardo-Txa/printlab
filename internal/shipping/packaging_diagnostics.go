package shipping

import (
	"encoding/json"
	"log"
)

// Bound output and accept only typed dimensions and positional indexes: never
// include names, identifiers, request/API bodies, CEPs or customer data.
const packagingDiagnosticLimit = 32

type productPackingDiagnostic struct {
	LineIndex    int    `json:"line_index"`
	Quantity     int    `json:"quantity"`
	WeightG      int64  `json:"weight_g"`
	DimensionsMM [3]int `json:"dimensions_hwl_mm"`
}

type boxFitDiagnostic struct {
	BoxIndex         int    `json:"box_index"`
	InternalMM       [3]int `json:"internal_hwl_mm"`
	InternalSortedMM [3]int `json:"internal_sorted_mm"`
	InternalValid    bool   `json:"internal_valid"`
	Fits             bool   `json:"fits"`
}

type packagingDiagnostic struct {
	ProductLines     int                        `json:"product_lines"`
	Units            int                        `json:"units"`
	Products         []productPackingDiagnostic `json:"products"`
	ProductsOmitted  int                        `json:"products_omitted"`
	CandidateBoxes   int                        `json:"candidate_boxes"`
	Boxes            []boxFitDiagnostic         `json:"boxes"`
	BoxesOmitted     int                        `json:"boxes_omitted"`
	PackingAlgorithm string                     `json:"packing_algorithm"`
}

func logNoFittingBoxDiagnostics(items []QuoteProduct, boxes []ShippingBox) {
	report := packagingDiagnostic{
		ProductLines:     len(items),
		CandidateBoxes:   len(boxes),
		Products:         make([]productPackingDiagnostic, 0),
		Boxes:            make([]boxFitDiagnostic, 0),
		PackingAlgorithm: "deterministic_extreme_points",
	}
	for i, item := range items {
		report.Units += item.Quantity
		if len(report.Products) >= packagingDiagnosticLimit {
			report.ProductsOmitted++
			continue
		}
		report.Products = append(report.Products, productPackingDiagnostic{
			LineIndex: i,
			Quantity:  item.Quantity,
			WeightG:   item.Profile.WeightG,
			DimensionsMM: [3]int{
				item.Profile.Dimensions.Height,
				item.Profile.Dimensions.Width,
				item.Profile.Dimensions.Length,
			},
		})
	}
	candidates := append([]ShippingBox(nil), boxes...)
	sortShippingBoxes(candidates)
	for i, box := range candidates {
		if len(report.Boxes) >= packagingDiagnosticLimit {
			report.BoxesOmitted++
			continue
		}
		report.Boxes = append(report.Boxes, boxFitDiagnostic{
			BoxIndex:         i,
			InternalMM:       [3]int{box.Internal.Height, box.Internal.Width, box.Internal.Length},
			InternalSortedMM: sortedDimensions(box.Internal),
			InternalValid:    box.Internal.Valid(),
			Fits:             FitsProductsInBox(items, box.Internal),
		})
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		return
	}
	log.Printf("shipping packaging diagnostic reason=no_fitting_box selection=real_box_packing data=%s", encoded)
}
