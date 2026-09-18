package admin

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

type activeProductFormRepository struct {
	*imageTestRepository
}

func (r *activeProductFormRepository) GetAdminProductForm(context.Context, string) (AdminProductFormPage, error) {
	return AdminProductFormPage{Form: AdminProductForm{IsActive: true}}, nil
}

func TestNewAdminProductStartsInactive(t *testing.T) {
	repo := &activeProductFormRepository{newImageTestRepository()}
	service := NewService(fakeAuthClient{}, repo, testAdminUserID, CookieOptions{})
	page, err := service.NewAdminProduct(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if page.Form.IsActive {
		t.Fatal("new products must start inactive")
	}
}

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
	tests := map[string]string{
		"Dragão Articulado":      "dragao-articulado",
		"PLA Branco Premium":     "pla-branco-premium",
		"  Caixa 20 x 15 x 10  ": "caixa-20-x-15-x-10",
		"Azul Céu":               "azul-ceu",
		" Suporte Ágil 3D! ":     "suporte-agil-3d",
		"foo---bar / baz":        "foo-bar-baz",
		"!!!":                    "",
	}
	for input, want := range tests {
		if got := CanonicalSlug(input); got != want {
			t.Errorf("CanonicalSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAutomaticSlugValidationUsesNameAndIgnoresSubmittedSlug(t *testing.T) {
	input, errorsByField := validateAdminProductForm(AdminProductForm{
		Name:     "Dragão Articulado",
		Slug:     "slug-malicioso",
		PriceBRL: "39,90",
	}, true, "")
	if errorsByField.Any() {
		t.Fatalf("unexpected errors: %#v", errorsByField)
	}
	if input.Slug != "dragao-articulado" {
		t.Fatalf("slug = %q, want server-generated slug", input.Slug)
	}
}

func TestAutomaticSlugValidationCoversAllCatalogEntities(t *testing.T) {
	productID := "11111111-1111-1111-1111-111111111111"
	tests := []struct {
		name  string
		check func() (string, AdminFieldErrors)
		want  string
	}{
		{name: "product", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminProductForm(AdminProductForm{Name: "Produto Ágil", PriceBRL: "1,00"}, true, "")
			return input.Slug, errs
		}, want: "produto-agil"},
		{name: "category", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminCategoryForm(AdminCategoryForm{Name: "Peças Azuis"}, true, "")
			return input.Slug, errs
		}, want: "pecas-azuis"},
		{name: "variant", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminVariantForm(AdminVariantForm{Name: "Tamanho Grande", SortOrder: "0"}, true, productID, "")
			return input.Slug, errs
		}, want: "tamanho-grande"},
		{name: "material", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminMaterialForm(AdminMaterialForm{Name: "PLA Branco"}, true, "")
			return input.Slug, errs
		}, want: "pla-branco"},
		{name: "color", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminColorForm(AdminColorForm{Name: "Azul Céu", HexColor: "#00AEEF"}, true, "")
			return input.Slug, errs
		}, want: "azul-ceu"},
		{name: "box", check: func() (string, AdminFieldErrors) {
			input, errs := validateAdminBoxForm(AdminBoxForm{Name: "Caixa Média", HeightMM: "12", WidthMM: "12", LengthMM: "12", PackagingWeightG: "100", SortOrder: "0"}, true, "")
			return input.Slug, errs
		}, want: "caixa-media"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, errs := test.check()
			if errs.Any() {
				t.Fatalf("unexpected errors: %#v", errs)
			}
			if got != test.want {
				t.Fatalf("slug = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAutomaticSlugCollisionCandidatesAreDeterministic(t *testing.T) {
	var attempts []string
	got, err := createWithAutomaticSlug("dragao-articulado", func(slug string) (string, error) {
		attempts = append(attempts, slug)
		if len(attempts) < 3 {
			return "", ErrDuplicateSlug
		}
		return "product-id", nil
	})
	if err != nil || got != "product-id" {
		t.Fatalf("create result = %q, %v", got, err)
	}
	want := []string{"dragao-articulado", "dragao-articulado-2", "dragao-articulado-3"}
	if !reflect.DeepEqual(attempts, want) {
		t.Fatalf("attempts = %#v, want %#v", attempts, want)
	}
}

func TestAutomaticSlugDoesNotMaskOtherUniqueErrors(t *testing.T) {
	wantErr := ErrDuplicateSKU
	var attempts int
	_, err := createWithAutomaticSlug("material", func(string) (string, error) {
		attempts++
		return "", wantErr
	})
	if !errors.Is(err, wantErr) || attempts != 1 {
		t.Fatalf("result = %v after %d attempts, want %v after one attempt", err, attempts, wantErr)
	}
}

func TestValidateAdminProductFormRejectsPartialShippingProfile(t *testing.T) {
	form := AdminProductForm{
		Name:            "Produto",
		Slug:            "produto",
		PriceBRL:        "39,90",
		IsActive:        true,
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

func TestValidateAdminProductShippingProfileRules(t *testing.T) {
	tests := []struct {
		name    string
		active  bool
		weight  string
		height  string
		width   string
		length  string
		wantErr bool
		wantNil bool
	}{
		{name: "inactive without profile", wantNil: true},
		{name: "inactive complete", weight: "100", height: "10", width: "20", length: "30"},
		{name: "inactive partial", weight: "100", wantErr: true},
		{name: "active complete", active: true, weight: "100", height: "10", width: "20", length: "30"},
		{name: "active without profile", active: true, wantErr: true},
		{name: "active partial", active: true, weight: "100", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errorsByField := AdminFieldErrors{}
			profile := validateAdminProductShippingProfile(errorsByField, test.active, test.weight, test.height, test.width, test.length)
			if errorsByField.Any() != test.wantErr {
				t.Fatalf("errors = %#v, wantErr %v", errorsByField, test.wantErr)
			}
			if (profile == nil) != test.wantNil {
				t.Fatalf("profile = %#v, wantNil %v", profile, test.wantNil)
			}
		})
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

func TestPrepareAdminVariantListItemsLabelsSingleActiveConfiguration(t *testing.T) {
	items := PrepareAdminVariantListItems([]AdminVariantListItem{
		{Name: "teste2", IsActive: true, IsDefault: false},
	})

	if items[0].DefaultLabel != "Única configuração" {
		t.Fatalf("expected single configuration label, got %q", items[0].DefaultLabel)
	}
}

func TestPrepareAdminVariantListItemsLabelsMultipleConfigurations(t *testing.T) {
	items := PrepareAdminVariantListItems([]AdminVariantListItem{
		{Name: "Mini", IsActive: true, IsDefault: true},
		{Name: "Grande", IsActive: true, IsDefault: false},
	})

	if items[0].DefaultLabel != "Padrão" {
		t.Fatalf("expected default label, got %q", items[0].DefaultLabel)
	}
	if items[1].DefaultLabel != "Não padrão" {
		t.Fatalf("expected non-default label, got %q", items[1].DefaultLabel)
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

func TestValidateAdminBoxFormRejectsInvalidOperationalDimension(t *testing.T) {
	form := AdminBoxForm{
		Name:             "Caixa",
		Slug:             "caixa",
		HeightMM:         "0",
		WidthMM:          "100",
		LengthMM:         "100",
		PackagingWeightG: "10",
		SortOrder:        "0",
	}

	_, errorsByField := validateAdminBoxForm(form, true, "")
	if errorsByField.Get("height_mm") == "" {
		t.Fatalf("expected height error, got %#v", errorsByField)
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
