package orders

import (
	"testing"
	"time"
)

func TestMaskCPFShowsOnlyLastTwoDigits(t *testing.T) {
	if got := MaskCPF("52998224725"); got != "***.***.***-25" {
		t.Fatalf("expected masked CPF, got %q", got)
	}
	if got := MaskCPF(""); got != "***.***.***-**" {
		t.Fatalf("expected empty CPF mask, got %q", got)
	}
}

func TestReviewFingerprintChangesWhenStateChanges(t *testing.T) {
	page := reviewPageFixtureForFingerprint()
	hash := bytes32ForTest("shipping-input")

	first, err := finalizeReviewPage(page, hash)
	if err != nil {
		t.Fatalf("expected first fingerprint, got %v", err)
	}

	page.Items[0].UnitPriceCents = 4990
	page.Items[0].LineTotalCents = 9980
	page.ProductsSubtotalCents = 9980
	page.TotalCents = 11870
	second, err := finalizeReviewPage(page, hash)
	if err != nil {
		t.Fatalf("expected second fingerprint, got %v", err)
	}

	if first.Fingerprint == "" || second.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprints")
	}
	if first.Fingerprint == second.Fingerprint {
		t.Fatal("expected fingerprint to change when price changes")
	}
}

func TestLineTotalRejectsOverflow(t *testing.T) {
	if _, err := lineTotalCents(9223372036854775807, 2); err == nil {
		t.Fatal("expected line total overflow")
	}
}

func TestOrderStatusAndNumberLabels(t *testing.T) {
	if got := StatusLabel(StatusPendingPayment); got != "Aguardando pagamento" {
		t.Fatalf("expected pending payment label, got %q", got)
	}
	if got := StatusLabel(StatusPaid); got != "Pagamento confirmado" {
		t.Fatalf("expected paid label, got %q", got)
	}
	if got := OrderNumberLabel(1001); got != "#1001" {
		t.Fatalf("expected order number label, got %q", got)
	}
}

func reviewPageFixtureForFingerprint() ReviewPage {
	printTime := 90
	weight := int64(12000)
	deliveryDays := 5
	quotedAt := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	return ReviewPage{
		Items: []ReviewItem{
			{
				CartItemID:                    "item-1",
				ProductID:                     "33333333-3333-3333-3333-333333333333",
				VariantID:                     "44444444-4444-4444-4444-444444444444",
				ProductName:                   "Produto Real",
				ProductSlug:                   "produto-real",
				VariantName:                   "Padrao",
				VariantSlug:                   "padrao",
				SKU:                           "PR-001",
				HasVariant:                    true,
				Quantity:                      2,
				UnitPriceCents:                3990,
				LineTotalCents:                7980,
				UnitPrintTimeMinutes:          &printTime,
				UnitEstimatedFilamentWeightMg: &weight,
				Filaments: []ReviewItemFilament{
					{
						MaterialName:             "PLA",
						MaterialSlug:             "pla",
						ColorName:                "Azul",
						ColorSlug:                "azul",
						HexColor:                 "#1187F4",
						EstimatedWeightMgPerUnit: 12000,
						Label:                    "Cor principal",
					},
				},
			},
		},
		Customer: ReviewCustomer{
			FullName: "Joao Silva",
			Email:    "joao@example.com",
			Phone:    "+5527999999999",
			CPF:      "52998224725",
		},
		Address: ReviewAddress{
			PostalCode:  "29100000",
			Street:      "Rua Um",
			Number:      "12A",
			Complement:  "Apto 302",
			District:    "Centro",
			City:        "Vila Velha",
			State:       "ES",
			CountryCode: "BR",
		},
		Shipping: ReviewShipping{
			Provider:         "superfrete",
			ServiceCode:      "1",
			ServiceName:      "PAC",
			CarrierName:      "Correios",
			DeliveryTimeDays: &deliveryDays,
			ShippingBoxName:  "Caixa Media",
			PriceCents:       1890,
			PackageWeightG:   720,
			PackageHeightMM:  120,
			PackageWidthMM:   160,
			PackageLengthMM:  240,
			QuotedAt:         &quotedAt,
		},
		ProductsSubtotalCents: 7980,
		ShippingPriceCents:    1890,
		TotalCents:            9870,
	}
}

func bytes32ForTest(seed string) []byte {
	output := make([]byte, 32)
	copy(output, seed)
	return output
}
