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

type packingCuboid struct {
	Height int
	Width  int
	Length int
	Volume int64
	Order  int
}

type packingPoint struct {
	X int
	Y int
	Z int
}

type packingPlacement struct {
	X      int
	Y      int
	Z      int
	Height int
	Width  int
	Length int
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

	sortShippingBoxes(candidates)
	return candidates[0], nil
}

func SelectShippingBoxForProducts(items []QuoteProduct, boxes []ShippingBox) (ShippingBox, error) {
	cuboids, err := packingCuboids(items)
	if err != nil {
		return ShippingBox{}, err
	}
	if len(boxes) == 0 {
		return ShippingBox{}, ErrNoFittingBox
	}

	candidates := append([]ShippingBox(nil), boxes...)
	sortShippingBoxes(candidates)
	for _, box := range candidates {
		if fitsProductsInBox(cuboids, box.Internal) {
			return box, nil
		}
	}

	return ShippingBox{}, ErrNoFittingBox
}

func FitsProductsInBox(items []QuoteProduct, boxDimensions DimensionsMM) bool {
	cuboids, err := packingCuboids(items)
	if err != nil {
		return false
	}
	return fitsProductsInBox(cuboids, boxDimensions)
}

func sortShippingBoxes(boxes []ShippingBox) {
	sort.SliceStable(boxes, func(i int, j int) bool {
		leftInternalVolume, leftInternalOverflow := boxVolume(boxes[i].Internal)
		rightInternalVolume, rightInternalOverflow := boxVolume(boxes[j].Internal)
		if leftInternalOverflow != rightInternalOverflow {
			return !leftInternalOverflow
		}
		if leftInternalVolume != rightInternalVolume {
			return leftInternalVolume < rightInternalVolume
		}
		if boxes[i].SortOrder != boxes[j].SortOrder {
			return boxes[i].SortOrder < boxes[j].SortOrder
		}
		leftExternalVolume, leftExternalOverflow := boxVolume(boxes[i].External)
		rightExternalVolume, rightExternalOverflow := boxVolume(boxes[j].External)
		if leftExternalOverflow != rightExternalOverflow {
			return !leftExternalOverflow
		}
		if leftExternalVolume != rightExternalVolume {
			return leftExternalVolume < rightExternalVolume
		}
		if boxes[i].PackagingWeightG != boxes[j].PackagingWeightG {
			return boxes[i].PackagingWeightG < boxes[j].PackagingWeightG
		}
		if boxes[i].Name != boxes[j].Name {
			return boxes[i].Name < boxes[j].Name
		}
		if boxes[i].Slug != boxes[j].Slug {
			return boxes[i].Slug < boxes[j].Slug
		}
		return boxes[i].ID < boxes[j].ID
	})
}

func packingCuboids(items []QuoteProduct) ([]packingCuboid, error) {
	var cuboids []packingCuboid
	order := 0
	for _, item := range items {
		if item.Quantity <= 0 || !item.Profile.Valid() {
			return nil, ErrMissingShippingProfile
		}
		volume, overflow := boxVolume(item.Profile.Dimensions)
		if overflow {
			return nil, ErrInvalidPackage
		}
		for unit := 0; unit < item.Quantity; unit++ {
			cuboids = append(cuboids, packingCuboid{
				Height: item.Profile.Dimensions.Height,
				Width:  item.Profile.Dimensions.Width,
				Length: item.Profile.Dimensions.Length,
				Volume: volume,
				Order:  order,
			})
			order++
		}
	}
	if len(cuboids) == 0 {
		return nil, ErrEmptyCart
	}

	sort.SliceStable(cuboids, func(i int, j int) bool {
		if cuboids[i].Volume != cuboids[j].Volume {
			return cuboids[i].Volume > cuboids[j].Volume
		}
		leftAxes := sortedDimensions(DimensionsMM{Height: cuboids[i].Height, Width: cuboids[i].Width, Length: cuboids[i].Length})
		rightAxes := sortedDimensions(DimensionsMM{Height: cuboids[j].Height, Width: cuboids[j].Width, Length: cuboids[j].Length})
		for axis := len(leftAxes) - 1; axis >= 0; axis-- {
			if leftAxes[axis] != rightAxes[axis] {
				return leftAxes[axis] > rightAxes[axis]
			}
		}
		return cuboids[i].Order < cuboids[j].Order
	})

	return cuboids, nil
}

func fitsProductsInBox(cuboids []packingCuboid, box DimensionsMM) bool {
	if !box.Valid() {
		return false
	}

	placements := make([]packingPlacement, 0, len(cuboids))
	points := []packingPoint{{}}
	for _, cuboid := range cuboids {
		placed := false
		sortPackingPoints(points)
		rotations := cuboidRotations(cuboid)
		for _, point := range points {
			for _, rotated := range rotations {
				placement := packingPlacement{X: point.X, Y: point.Y, Z: point.Z, Height: rotated.Height, Width: rotated.Width, Length: rotated.Length}
				if placementFits(placement, box, placements) {
					placements = append(placements, placement)
					points = append(points,
						packingPoint{X: placement.X + placement.Length, Y: placement.Y, Z: placement.Z},
						packingPoint{X: placement.X, Y: placement.Y + placement.Width, Z: placement.Z},
						packingPoint{X: placement.X, Y: placement.Y, Z: placement.Z + placement.Height},
					)
					points = normalizePackingPoints(points, box, placements)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			return false
		}
	}

	return true
}

func cuboidRotations(c packingCuboid) []DimensionsMM {
	values := []DimensionsMM{
		{Height: c.Height, Width: c.Width, Length: c.Length},
		{Height: c.Height, Width: c.Length, Length: c.Width},
		{Height: c.Width, Width: c.Height, Length: c.Length},
		{Height: c.Width, Width: c.Length, Length: c.Height},
		{Height: c.Length, Width: c.Height, Length: c.Width},
		{Height: c.Length, Width: c.Width, Length: c.Height},
	}
	rotations := make([]DimensionsMM, 0, len(values))
	seen := map[DimensionsMM]bool{}
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		rotations = append(rotations, value)
	}
	sort.SliceStable(rotations, func(i int, j int) bool {
		if rotations[i].Height != rotations[j].Height {
			return rotations[i].Height < rotations[j].Height
		}
		if rotations[i].Width != rotations[j].Width {
			return rotations[i].Width < rotations[j].Width
		}
		return rotations[i].Length < rotations[j].Length
	})
	return rotations
}

func placementFits(candidate packingPlacement, box DimensionsMM, placed []packingPlacement) bool {
	if candidate.Height <= 0 || candidate.Width <= 0 || candidate.Length <= 0 {
		return false
	}
	if candidate.X < 0 || candidate.Y < 0 || candidate.Z < 0 {
		return false
	}
	if candidate.X+candidate.Length > box.Length || candidate.Y+candidate.Width > box.Width || candidate.Z+candidate.Height > box.Height {
		return false
	}
	for _, existing := range placed {
		if placementsOverlap(candidate, existing) {
			return false
		}
	}
	return true
}

func placementsOverlap(a packingPlacement, b packingPlacement) bool {
	return a.X < b.X+b.Length && a.X+a.Length > b.X &&
		a.Y < b.Y+b.Width && a.Y+a.Width > b.Y &&
		a.Z < b.Z+b.Height && a.Z+a.Height > b.Z
}

func normalizePackingPoints(points []packingPoint, box DimensionsMM, placed []packingPlacement) []packingPoint {
	seen := map[packingPoint]bool{}
	filtered := make([]packingPoint, 0, len(points))
	for _, point := range points {
		if point.X < 0 || point.Y < 0 || point.Z < 0 || point.X >= box.Length || point.Y >= box.Width || point.Z >= box.Height {
			continue
		}
		insidePlaced := false
		for _, placement := range placed {
			if point.X >= placement.X && point.X < placement.X+placement.Length &&
				point.Y >= placement.Y && point.Y < placement.Y+placement.Width &&
				point.Z >= placement.Z && point.Z < placement.Z+placement.Height {
				insidePlaced = true
				break
			}
		}
		if insidePlaced || seen[point] {
			continue
		}
		seen[point] = true
		filtered = append(filtered, point)
	}
	sortPackingPoints(filtered)
	return filtered
}

func sortPackingPoints(points []packingPoint) {
	sort.SliceStable(points, func(i int, j int) bool {
		if points[i].Z != points[j].Z {
			return points[i].Z < points[j].Z
		}
		if points[i].Y != points[j].Y {
			return points[i].Y < points[j].Y
		}
		return points[i].X < points[j].X
	})
}

func BuildCartPackageInputHash(originCEP string, destinationCEP string, services []string, items []CartItem, packageSnapshot ShippingPackage) ([]byte, error) {
	preparedItems, err := prepareQuoteProducts(items)
	if err != nil {
		return nil, err
	}
	return BuildInputHash(quoteFingerprint(originCEP, destinationCEP, services, preparedItems, packageSnapshot))
}

func FallbackPackageForCartItems(items []CartItem) (ShippingPackage, error) {
	preparedItems, err := prepareQuoteProducts(items)
	if err != nil {
		return ShippingPackage{}, err
	}
	return FallbackPackageForProducts(preparedItems)
}

func FallbackPackageForProducts(items []QuoteProduct) (ShippingPackage, error) {
	units, goodsWeightG, err := fallbackUnits(items)
	if err != nil {
		return ShippingPackage{}, err
	}

	var height, width, length int
	for _, unit := range units {
		if unit.Height > height {
			height = unit.Height
		}
		if unit.Width > width {
			width = unit.Width
		}
		if length > math.MaxInt-unit.Length {
			return ShippingPackage{}, ErrInvalidPackage
		}
		length += unit.Length
	}
	height = ceilToMultipleInt(height+20, 10)
	width = ceilToMultipleInt(width+20, 10)
	length = ceilToMultipleInt(length+20, 10)

	packagingWeightG := fallbackPackagingWeightG(goodsWeightG, len(units))
	if goodsWeightG > math.MaxInt64-packagingWeightG {
		return ShippingPackage{}, ErrAmountOverflow
	}
	weightG := ceilToMultipleInt64(goodsWeightG+packagingWeightG, 50)
	return ShippingPackage{
		PackagingSource:  PackagingSourceFallback,
		PackagingWeightG: packagingWeightG,
		WeightG:          weightG,
		Dimensions:       DimensionsMM{Height: height, Width: width, Length: length},
	}, nil
}

func fallbackUnits(items []QuoteProduct) ([]DimensionsMM, int64, error) {
	var units []DimensionsMM
	var goodsWeightG int64
	for _, item := range items {
		if item.Quantity <= 0 || !item.Profile.Valid() {
			return nil, 0, ErrMissingShippingProfile
		}
		if item.Profile.WeightG > math.MaxInt64/int64(item.Quantity) {
			return nil, 0, ErrAmountOverflow
		}
		lineWeight := item.Profile.WeightG * int64(item.Quantity)
		if goodsWeightG > math.MaxInt64-lineWeight {
			return nil, 0, ErrAmountOverflow
		}
		goodsWeightG += lineWeight
		axes := sortedDimensions(item.Profile.Dimensions)
		unit := DimensionsMM{Height: axes[0] + 20, Width: axes[1] + 20, Length: axes[2] + 20}
		for i := 0; i < item.Quantity; i++ {
			units = append(units, unit)
		}
	}
	if len(units) == 0 {
		return nil, 0, ErrEmptyCart
	}
	return units, goodsWeightG, nil
}

func fallbackPackagingWeightG(goodsWeightG int64, units int) int64 {
	byPercent := ceilDivInt64(goodsWeightG*25, 100) // ceil 25% to whole grams.
	byUnit := int64(50 * units)
	weight := int64(150)
	if byPercent > weight {
		weight = byPercent
	}
	if byUnit > weight {
		weight = byUnit
	}
	return ceilToMultipleInt64(weight, 50)
}

func ceilToMultipleInt(value int, multiple int) int {
	if multiple <= 0 || value <= 0 {
		return value
	}
	return ((value + multiple - 1) / multiple) * multiple
}

func ceilToMultipleInt64(value int64, multiple int64) int64 {
	if multiple <= 0 || value <= 0 {
		return value
	}
	return ((value + multiple - 1) / multiple) * multiple
}

func ceilDivInt64(value int64, divisor int64) int64 {
	if divisor <= 0 || value <= 0 {
		return value
	}
	return (value + divisor - 1) / divisor
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
		return "Prazo indisponível"
	}
	if *days == 1 {
		return "1 dia útil"
	}

	return strconv.Itoa(*days) + " dias úteis"
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

func BuildCartInputHash(originCEP string, destinationCEP string, services []string, items []CartItem, box ShippingBox) ([]byte, error) {
	return BuildCartPackageInputHash(originCEP, destinationCEP, services, items, ShippingPackage{
		Box:              box,
		PackagingSource:  PackagingSourceRealBox,
		PackagingWeightG: box.PackagingWeightG,
		Dimensions:       box.External,
	})
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
