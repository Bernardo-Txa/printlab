package orders

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
)

type reviewFingerprint struct {
	Items                 []reviewFingerprintItem   `json:"items"`
	Customer              reviewFingerprintCustomer `json:"customer"`
	Address               reviewFingerprintAddress  `json:"address"`
	Shipping              reviewFingerprintShipping `json:"shipping"`
	ProductsSubtotalCents int64                     `json:"products_subtotal_cents"`
	ShippingPriceCents    int64                     `json:"shipping_price_cents"`
	TotalCents            int64                     `json:"total_cents"`
}

type reviewFingerprintItem struct {
	CartItemID       string `json:"cart_item_id"`
	ProductID        string `json:"product_id"`
	VariantID        string `json:"variant_id"`
	Quantity         int    `json:"quantity"`
	UnitPriceCents   int64  `json:"unit_price_cents"`
	LineTotalCents   int64  `json:"line_total_cents"`
	PrintTimeMinutes *int   `json:"unit_print_time_minutes,omitempty"`
	FilamentWeightMg *int64 `json:"unit_estimated_filament_weight_mg,omitempty"`
}

type reviewFingerprintCustomer struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	CPF      string `json:"cpf"`
}

type reviewFingerprintAddress struct {
	PostalCode  string `json:"postal_code"`
	Street      string `json:"street"`
	Number      string `json:"number"`
	Complement  string `json:"complement"`
	District    string `json:"district"`
	City        string `json:"city"`
	State       string `json:"state"`
	CountryCode string `json:"country_code"`
}

type reviewFingerprintShipping struct {
	Provider         string `json:"provider"`
	ServiceCode      string `json:"service_code"`
	ServiceName      string `json:"service_name"`
	CarrierName      string `json:"carrier_name"`
	DeliveryTimeDays *int   `json:"delivery_time_days,omitempty"`
	BoxName          string `json:"shipping_box_name"`
	PackageWeightG   int64  `json:"package_weight_g"`
	PackageHeightMM  int    `json:"package_height_mm"`
	PackageWidthMM   int    `json:"package_width_mm"`
	PackageLengthMM  int    `json:"package_length_mm"`
	InputHash        string `json:"input_hash"`
	PriceCents       int64  `json:"price_cents"`
}

func finalizeReviewPage(page ReviewPage, inputHash []byte) (ReviewPage, error) {
	page.ProductsSubtotalBRL = products.FormatBRL(page.ProductsSubtotalCents)
	page.ShippingPriceBRL = products.FormatBRL(page.ShippingPriceCents)
	page.TotalBRL = products.FormatBRL(page.TotalCents)
	page.Customer.MaskedCPF = MaskCPF(page.Customer.CPF)
	page.Address = prepareAddress(page.Address)
	page.Shipping.PriceBRL = products.FormatBRL(page.Shipping.PriceCents)
	page.Shipping.DeliveryTime = shipping.DeliveryTimeLabel(page.Shipping.DeliveryTimeDays)

	for i := range page.Items {
		page.Items[i].UnitPriceBRL = products.FormatBRL(page.Items[i].UnitPriceCents)
		page.Items[i].LineTotalBRL = products.FormatBRL(page.Items[i].LineTotalCents)
		if page.Items[i].UnitEstimatedFilamentWeightMg != nil {
			page.Items[i].UnitEstimatedFilamentWeightLabel = formatWeightMg(*page.Items[i].UnitEstimatedFilamentWeightMg)
		}
		if page.Items[i].UnitPrintTimeMinutes != nil || len(page.Items[i].Filaments) > 0 {
			page.HasProductionSnapshots = true
		}
		for j := range page.Items[i].Filaments {
			page.Items[i].Filaments[j].EstimatedWeightLabel = formatWeightMg(page.Items[i].Filaments[j].EstimatedWeightMgPerUnit)
		}
	}

	fingerprint, err := buildReviewFingerprint(page, inputHash)
	if err != nil {
		return ReviewPage{}, err
	}
	page.Fingerprint = fingerprint

	return page, nil
}

func buildReviewFingerprint(page ReviewPage, inputHash []byte) (string, error) {
	payload := reviewFingerprint{
		Items: make([]reviewFingerprintItem, 0, len(page.Items)),
		Customer: reviewFingerprintCustomer{
			FullName: page.Customer.FullName,
			Email:    page.Customer.Email,
			Phone:    page.Customer.Phone,
			CPF:      page.Customer.CPF,
		},
		Address: reviewFingerprintAddress{
			PostalCode:  page.Address.PostalCode,
			Street:      page.Address.Street,
			Number:      page.Address.Number,
			Complement:  page.Address.Complement,
			District:    page.Address.District,
			City:        page.Address.City,
			State:       page.Address.State,
			CountryCode: page.Address.CountryCode,
		},
		Shipping: reviewFingerprintShipping{
			Provider:         page.Shipping.Provider,
			ServiceCode:      page.Shipping.ServiceCode,
			ServiceName:      page.Shipping.ServiceName,
			CarrierName:      page.Shipping.CarrierName,
			DeliveryTimeDays: page.Shipping.DeliveryTimeDays,
			BoxName:          page.Shipping.ShippingBoxName,
			PackageWeightG:   page.Shipping.PackageWeightG,
			PackageHeightMM:  page.Shipping.PackageHeightMM,
			PackageWidthMM:   page.Shipping.PackageWidthMM,
			PackageLengthMM:  page.Shipping.PackageLengthMM,
			InputHash:        hex.EncodeToString(inputHash),
			PriceCents:       page.Shipping.PriceCents,
		},
		ProductsSubtotalCents: page.ProductsSubtotalCents,
		ShippingPriceCents:    page.ShippingPriceCents,
		TotalCents:            page.TotalCents,
	}

	for _, item := range page.Items {
		payload.Items = append(payload.Items, reviewFingerprintItem{
			CartItemID:       item.CartItemID,
			ProductID:        item.ProductID,
			VariantID:        item.VariantID,
			Quantity:         item.Quantity,
			UnitPriceCents:   item.UnitPriceCents,
			LineTotalCents:   item.LineTotalCents,
			PrintTimeMinutes: item.UnitPrintTimeMinutes,
			FilamentWeightMg: item.UnitEstimatedFilamentWeightMg,
		})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func lineTotalCents(unitPriceCents int64, quantity int) (int64, error) {
	if unitPriceCents < 0 || quantity < 1 || quantity > 99 {
		return 0, ErrAmountOverflow
	}
	if quantity != 0 && unitPriceCents > math.MaxInt64/int64(quantity) {
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

func addWeightMg(left int64, right int64) (int64, error) {
	if left < 0 || right <= 0 || left > math.MaxInt64-right {
		return 0, ErrAmountOverflow
	}

	return left + right, nil
}

func MaskCPF(cpf string) string {
	digits := onlyDigits(cpf)
	if len(digits) < 2 {
		return "***.***.***-**"
	}

	return "***.***.***-" + digits[len(digits)-2:]
}

func StatusLabel(status string) string {
	if status == StatusPendingPayment {
		return "Aguardando pagamento"
	}

	return "Status indisponivel"
}

func OrderNumberLabel(orderNumber int64) string {
	return "#" + strconv.FormatInt(orderNumber, 10)
}

func ValidOrderID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}

	return true
}

func prepareAddress(address ReviewAddress) ReviewAddress {
	address.LineOne = address.Street + ", " + address.Number
	if address.Complement != "" {
		address.LineOne += " - " + address.Complement
	}
	address.LineTwo = address.District + " - " + address.City + "/" + address.State + " - CEP " + formatPostalCode(address.PostalCode)
	return address
}

func formatPostalCode(postalCode string) string {
	digits := onlyDigits(postalCode)
	if len(digits) != 8 {
		return postalCode
	}

	return digits[:5] + "-" + digits[5:]
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func formatWeightMg(weightMg int64) string {
	if weightMg > 0 && weightMg%1000 == 0 {
		return strconv.FormatInt(weightMg/1000, 10) + " g"
	}

	return strconv.FormatInt(weightMg, 10) + " mg"
}
