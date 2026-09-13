package admin

import (
	"errors"
	"math"
	"testing"
)

func TestParseAdminBRLCents(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "comma cents", value: "39,90", want: 3990},
		{name: "dot cents", value: "39.90", want: 3990},
		{name: "zero", value: "0,00", want: 0},
		{name: "integer", value: "1", want: 100},
		{name: "negative", value: "-1,00", wantErr: true},
		{name: "three decimals", value: "1,999", wantErr: true},
		{name: "scientific", value: "1e3", wantErr: true},
		{name: "overflow", value: "92233720368547758,08", wantErr: true},
		{name: "text", value: "abc", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseAdminBRLCents(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestParseAdminGramsToMilligrams(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "integer grams", value: "12", want: 12000},
		{name: "comma decimal", value: "12,5", want: 12500},
		{name: "dot decimal", value: "12.500", want: 12500},
		{name: "milligrams", value: "0,850", want: 850},
		{name: "more than three decimals", value: "1,0001", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "overflow", value: "9223372036854776", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseAdminGramsToMilligrams(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestCanonicalSlug(t *testing.T) {
	if got := CanonicalSlug(" Suporte Ágil 3D! "); got != "suporte-agil-3d" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestValidateAdminProductFormRejectsPartialShippingProfile(t *testing.T) {
	form := AdminProductForm{
		Name:            "Produto",
		Slug:            "produto",
		PriceBRL:        "39,90",
		UseShipping:     true,
		ShippingWeightG: "100",
	}

	_, errorsByField := validateAdminProductForm(form, true, "")
	if !errorsByField.Any() {
		t.Fatal("expected validation errors")
	}
	if errorsByField.Get("shipping_height_mm") == "" || errorsByField.Get("shipping_width_mm") == "" || errorsByField.Get("shipping_length_mm") == "" {
		t.Fatalf("expected missing dimensions to be rejected, got %#v", errorsByField)
	}
}

func TestValidateAdminVariantFormKeepsZeroPriceOverride(t *testing.T) {
	form := AdminVariantForm{
		Name:      "Padrao",
		Slug:      "padrao",
		PriceBRL:  "0,00",
		IsActive:  true,
		IsDefault: true,
		SortOrder: "0",
	}

	input, errorsByField := validateAdminVariantForm(form, true, "11111111-1111-1111-1111-111111111111", "")
	if errorsByField.Any() {
		t.Fatalf("unexpected errors: %#v", errorsByField)
	}
	if input.PriceCents == nil || *input.PriceCents != 0 {
		t.Fatalf("expected explicit zero override, got %#v", input.PriceCents)
	}
	if !input.IsDefault {
		t.Fatal("expected active default variant")
	}
}

func TestValidateAdminVariantFormCannotKeepInactiveDefault(t *testing.T) {
	form := AdminVariantForm{
		Name:      "Padrao",
		Slug:      "padrao",
		IsActive:  false,
		IsDefault: true,
		SortOrder: "0",
	}

	input, errorsByField := validateAdminVariantForm(form, true, "11111111-1111-1111-1111-111111111111", "")
	if errorsByField.Get("is_default") == "" {
		t.Fatalf("expected inactive default error, got %#v", errorsByField)
	}
	if input.IsDefault {
		t.Fatal("inactive variant must not remain default")
	}
}

func TestValidateAdminVariantFormDeactivatingDefaultClearsDefault(t *testing.T) {
	form := AdminVariantForm{
		Name:      "Padrao",
		Slug:      "padrao",
		IsActive:  false,
		IsDefault: true,
		SortOrder: "0",
	}

	input, errorsByField := validateAdminVariantForm(form, false, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")
	if errorsByField.Any() {
		t.Fatalf("unexpected errors: %#v", errorsByField)
	}
	if input.IsDefault {
		t.Fatal("deactivated variant must clear default")
	}
}

func TestNormalizeHexColor(t *testing.T) {
	got, err := NormalizeHexColor("#ff0000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "#FF0000" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeHexColor("red"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestValidateAdminBoxFormRejectsExternalSmallerThanInternal(t *testing.T) {
	form := AdminBoxForm{
		Name:             "Caixa",
		Slug:             "caixa",
		InternalHeightMM: "100",
		InternalWidthMM:  "100",
		InternalLengthMM: "100",
		ExternalHeightMM: "99",
		ExternalWidthMM:  "100",
		ExternalLengthMM: "100",
		PackagingWeightG: "10",
		SortOrder:        "0",
	}

	_, errorsByField := validateAdminBoxForm(form, true, "")
	if errorsByField.Get("external_height_mm") == "" {
		t.Fatalf("expected external height error, got %#v", errorsByField)
	}
}

func TestParseAdminNonNegativeIntRejectsOverflow(t *testing.T) {
	if _, err := parseAdminNonNegativeInt64("9223372036854775808"); err == nil {
		t.Fatal("expected overflow error")
	}
	if _, err := parseAdminNonNegativeInt(strconvValue(math.MaxInt64)); err == nil {
		t.Fatal("expected int overflow error")
	}
}

func strconvValue(value int64) string {
	return FormatAdminInt64(value)
}
