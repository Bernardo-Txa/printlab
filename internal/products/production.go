package products

import (
	"strconv"
	"strings"
)

func EffectivePriceCents(product Product, variant ProductVariant) int64 {
	if variant.PriceCents != nil {
		return *variant.PriceCents
	}

	return product.PriceCents
}

func TotalFilamentWeightMg(filaments []VariantFilament) int64 {
	var total int64
	for _, filament := range filaments {
		total += filament.EstimatedWeightMg
	}

	return total
}

func FormatWeightGrams(weightMg int64) string {
	grams := weightMg / 1000
	milligrams := weightMg % 1000
	if milligrams == 0 {
		return strconv.FormatInt(grams, 10) + " g"
	}

	decimals := strconv.FormatInt(milligrams+1000, 10)[1:]
	decimals = strings.TrimRight(decimals, "0")

	return strconv.FormatInt(grams, 10) + "," + decimals + " g"
}

func FormatPrintTime(minutes int) string {
	hours := minutes / 60
	remainingMinutes := minutes % 60

	switch {
	case hours > 0 && remainingMinutes > 0:
		return strconv.Itoa(hours) + "h " + strconv.Itoa(remainingMinutes) + "min"
	case hours > 0:
		return strconv.Itoa(hours) + "h"
	default:
		return strconv.Itoa(remainingMinutes) + "min"
	}
}
