package cart

import (
	"math"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

func lineSubtotalCents(unitPriceCents int64, quantity int) (int64, error) {
	if unitPriceCents < 0 || quantity < MinQuantity {
		return 0, ErrAmountOverflow
	}

	if unitPriceCents > math.MaxInt64/int64(quantity) {
		return 0, ErrAmountOverflow
	}

	return unitPriceCents * int64(quantity), nil
}

func addCents(left int64, right int64) (int64, error) {
	if left < 0 || right < 0 || left > math.MaxInt64-right {
		return 0, ErrAmountOverflow
	}

	return left + right, nil
}

func formatBRL(cents int64) string {
	return products.FormatBRL(cents)
}
