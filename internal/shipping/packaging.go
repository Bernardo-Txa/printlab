package shipping

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
)

func EffectiveShippingProfile(item CartItem) (ShippingProfile, bool) {
	if item.VariantProfile != nil && item.VariantProfile.Valid() {
		return *item.VariantProfile, true
	}
	if item.ProductProfile != nil && item.ProductProfile.Valid() {
		return *item.ProductProfile, true
	}

	return ShippingProfile{}, false
}

func (p ShippingProfile) Valid() bool {
	return p.WeightG > 0 && p.Dimensions.Valid()
}

func (d DimensionsMM) Valid() bool {
	return d.Height > 0 && d.Width > 0 && d.Length > 0
}

func FitsInside(packageDimensions DimensionsMM, boxDimensions DimensionsMM) bool {
	if !packageDimensions.Valid() || !boxDimensions.Valid() {
		return false
	}

	packageAxes := sortedDimensions(packageDimensions)
	boxAxes := sortedDimensions(boxDimensions)
	for i := range packageAxes {
		if packageAxes[i] > boxAxes[i] {
			return false
		}
	}

	return true
}

func SelectSmallestBox(packageDimensions DimensionsMM, boxes []ShippingBox) (ShippingBox, error) {
	if !packageDimensions.Valid() {
		return ShippingBox{}, ErrInvalidPackage
	}

	candidates := make([]ShippingBox, 0, len(boxes))
	for _, box := range boxes {
		if FitsInside(packageDimensions, box.Internal) {
			candidates = append(candidates, box)
		}
	}
	if len(candidates) == 0 {
		return ShippingBox{}, ErrNoFittingBox
	}

	sort.SliceStable(candidates, func(i int, j int) bool {
		leftVolume, leftOverflow := boxVolume(candidates[i].Internal)
		rightVolume, rightOverflow := boxVolume(candidates[j].Internal)
		if leftOverflow != rightOverflow {
			return !leftOverflow
		}
		if leftVolume != rightVolume {
			return leftVolume < rightVolume
		}
		if candidates[i].PackagingWeightG != candidates[j].PackagingWeightG {
			return candidates[i].PackagingWeightG < candidates[j].PackagingWeightG
		}
		if candidates[i].SortOrder != candidates[j].SortOrder {
			return candidates[i].SortOrder < candidates[j].SortOrder
		}
		if candidates[i].Name != candidates[j].Name {
			return candidates[i].Name < candidates[j].Name
		}

		return candidates[i].ID < candidates[j].ID
	})

	return candidates[0], nil
}

func TotalPackageWeightG(items []QuoteProduct, packagingWeightG int64) (int64, error) {
	if packagingWeightG <= 0 {
		return 0, ErrInvalidPackage
	}

	total := packagingWeightG
	for _, item := range items {
		if item.Quantity <= 0 || !item.Profile.Valid() {
			return 0, ErrMissingShippingProfile
		}
		if item.Profile.WeightG > math.MaxInt64/int64(item.Quantity) {
			return 0, ErrAmountOverflow
		}
		lineWeight := item.Profile.WeightG * int64(item.Quantity)
		if total > math.MaxInt64-lineWeight {
			return 0, ErrAmountOverflow
		}
		total += lineWeight
	}

	return total, nil
}

func GramsToKilograms(weightG int64) float64 {
	return float64(weightG) / 1000
}

func MillimetersToCentimeters(mm int) float64 {
	return float64(mm) / 10
}

func CentimetersToMillimetersCeil(value string) (int, error) {
	scaled, err := decimalToScaledCeil(value, 10)
	if err != nil || scaled <= 0 || scaled > int64(math.MaxInt) {
		return 0, ErrInvalidPackage
	}

	return int(scaled), nil
}

func DecimalToCents(value string) (int64, error) {
	cents, err := decimalToScaledRounded(value, 100)
	if err != nil || cents < 0 {
		return 0, ErrNoQuotes
	}

	return cents, nil
}

func FormatBRL(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}

	reais := cents / 100
	centavos := cents % 100
	digits := strconv.FormatInt(reais, 10)
	var groups []string
	for len(digits) > 3 {
		groups = append([]string{digits[len(digits)-3:]}, groups...)
		digits = digits[:len(digits)-3]
	}
	groups = append([]string{digits}, groups...)

	return "R$ " + sign + strings.Join(groups, ".") + "," + leftPad2(centavos)
}

func DeliveryTimeLabel(days *int) string {
	if days == nil {
		return "Prazo indisponivel"
	}
	if *days == 1 {
		return "1 dia util"
	}

	return strconv.Itoa(*days) + " dias uteis"
}

func PartialTotalBRL(productsSubtotalCents int64, shippingPriceCents int64) (string, error) {
	if shippingPriceCents < 0 || productsSubtotalCents > math.MaxInt64-shippingPriceCents {
		return "", ErrAmountOverflow
	}

	return FormatBRL(productsSubtotalCents + shippingPriceCents), nil
}

func BuildInputHash(fingerprint QuoteFingerprint) ([]byte, error) {
	canonical := fingerprint
	canonical.Products = append([]QuoteProductFingerprint(nil), fingerprint.Products...)
	canonical.Services = append([]string(nil), fingerprint.Services...)
	sort.Slice(canonical.Products, func(i int, j int) bool {
		if canonical.Products[i].ProductID != canonical.Products[j].ProductID {
			return canonical.Products[i].ProductID < canonical.Products[j].ProductID
		}
		if canonical.Products[i].VariantID != canonical.Products[j].VariantID {
			return canonical.Products[i].VariantID < canonical.Products[j].VariantID
		}
		return canonical.Products[i].ID < canonical.Products[j].ID
	})
	sort.Slice(canonical.Services, func(i int, j int) bool {
		return serviceSortValue(canonical.Services[i]) < serviceSortValue(canonical.Services[j])
	})

	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(payload)
	return hash[:], nil
}

func InputHashHex(hash []byte) string {
	return hex.EncodeToString(hash)
}

func sortedDimensions(dimensions DimensionsMM) [3]int {
	axes := [3]int{dimensions.Height, dimensions.Width, dimensions.Length}
	sort.Ints(axes[:])
	return axes
}

func boxVolume(dimensions DimensionsMM) (int64, bool) {
	axes := []int64{int64(dimensions.Height), int64(dimensions.Width), int64(dimensions.Length)}
	if axes[0] <= 0 || axes[1] <= 0 || axes[2] <= 0 {
		return 0, true
	}
	if axes[0] > math.MaxInt64/axes[1] {
		return 0, true
	}
	value := axes[0] * axes[1]
	if value > math.MaxInt64/axes[2] {
		return 0, true
	}

	return value * axes[2], false
}

func decimalToScaledCeil(value string, scale int64) (int64, error) {
	integer, fraction, err := parseDecimalParts(value)
	if err != nil {
		return 0, err
	}

	scaled, overflow := multiplyInt64(integer, scale)
	if overflow {
		return 0, ErrInvalidPackage
	}
	if fraction == "" {
		return scaled, nil
	}

	denominator := pow10(len(fraction))
	if denominator == 0 {
		return 0, ErrInvalidPackage
	}
	numerator, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	if numerator == 0 {
		return scaled, nil
	}

	fractionScaled := numerator * scale / denominator
	if numerator*scale%denominator != 0 {
		fractionScaled++
	}
	if scaled > math.MaxInt64-fractionScaled {
		return 0, ErrInvalidPackage
	}

	return scaled + fractionScaled, nil
}

func decimalToScaledRounded(value string, scale int64) (int64, error) {
	integer, fraction, err := parseDecimalParts(value)
	if err != nil {
		return 0, err
	}

	scaled, overflow := multiplyInt64(integer, scale)
	if overflow {
		return 0, ErrNoQuotes
	}
	if fraction == "" {
		return scaled, nil
	}

	denominator := pow10(len(fraction))
	if denominator == 0 {
		return 0, ErrNoQuotes
	}
	numerator, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}

	fractionScaled := numerator * scale / denominator
	remainder := numerator * scale % denominator
	if remainder*2 >= denominator {
		fractionScaled++
	}
	if scaled > math.MaxInt64-fractionScaled {
		return 0, ErrNoQuotes
	}

	return scaled + fractionScaled, nil
}

func parseDecimalParts(value string) (int64, string, error) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	if value == "" || strings.HasPrefix(value, "-") || strings.Contains(value, ",") {
		return 0, "", ErrInvalidPackage
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, "", ErrInvalidPackage
	}
	if !asciiDigits(parts[0]) {
		return 0, "", ErrInvalidPackage
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || !asciiDigits(fraction) {
			return 0, "", ErrInvalidPackage
		}
	}

	integer, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}

	return integer, fraction, nil
}

func asciiDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func multiplyInt64(left int64, right int64) (int64, bool) {
	if left > 0 && right > math.MaxInt64/left {
		return 0, true
	}

	return left * right, false
}

func pow10(digits int) int64 {
	if digits < 0 || digits > 18 {
		return 0
	}

	value := int64(1)
	for range digits {
		if value > math.MaxInt64/10 {
			return 0
		}
		value *= 10
	}

	return value
}

func serviceSortValue(value string) int {
	code, err := strconv.Atoi(value)
	if err != nil {
		return math.MaxInt
	}

	return code
}

func inputHashEqual(left []byte, right []byte) bool {
	return bytes.Equal(left, right)
}

func leftPad2(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}

	return strconv.FormatInt(value, 10)
}
