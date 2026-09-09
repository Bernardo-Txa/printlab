package products

import (
	"fmt"
	"strconv"
)

func FormatBRL(cents int64) string {
	reais := cents / 100
	centavos := cents % 100

	return fmt.Sprintf("R$ %s,%02d", formatThousands(reais), centavos)
}

func formatThousands(value int64) string {
	digits := strconv.FormatInt(value, 10)
	if len(digits) <= 3 {
		return digits
	}

	firstGroupSize := len(digits) % 3
	if firstGroupSize == 0 {
		firstGroupSize = 3
	}

	result := digits[:firstGroupSize]
	for i := firstGroupSize; i < len(digits); i += 3 {
		result += "." + digits[i:i+3]
	}

	return result
}
