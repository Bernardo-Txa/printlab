package customers

import "testing"

func TestNormalizeCPFValidFormatted(t *testing.T) {
	got, err := NormalizeCPF("529.982.247-25")
	if err != nil {
		t.Fatalf("expected valid CPF, got %v", err)
	}
	if got != "52998224725" {
		t.Fatalf("expected normalized CPF, got %q", got)
	}
}

func TestNormalizeCPFValidUnmasked(t *testing.T) {
	got, err := NormalizeCPF("11144477735")
	if err != nil {
		t.Fatalf("expected valid CPF, got %v", err)
	}
	if got != "11144477735" {
		t.Fatalf("expected normalized CPF, got %q", got)
	}
}

func TestNormalizeCPFRejectsInvalidCheckDigits(t *testing.T) {
	for _, input := range []string{
		"529.982.247-15",
		"529.982.247-24",
	} {
		if _, err := NormalizeCPF(input); err == nil {
			t.Fatalf("expected invalid CPF %q to fail", input)
		}
	}
}

func TestNormalizeCPFRejectsInvalidLengthLettersAndWhitespaceOnly(t *testing.T) {
	for _, input := range []string{
		"5299822472",
		"529982247255",
		"529abc24725",
		"   ",
	} {
		if _, err := NormalizeCPF(input); err == nil {
			t.Fatalf("expected invalid CPF %q to fail", input)
		}
	}
}

func TestNormalizeCPFRejectsAllRepeatedSequences(t *testing.T) {
	for digit := '0'; digit <= '9'; digit++ {
		input := string([]rune{digit, digit, digit, digit, digit, digit, digit, digit, digit, digit, digit})
		if _, err := NormalizeCPF(input); err == nil {
			t.Fatalf("expected repeated CPF %q to fail", input)
		}
	}
}

func TestNormalizeCPFTrimsWhitespace(t *testing.T) {
	got, err := NormalizeCPF(" 529.982.247-25 ")
	if err != nil {
		t.Fatalf("expected valid CPF with whitespace, got %v", err)
	}
	if got != "52998224725" {
		t.Fatalf("expected normalized CPF, got %q", got)
	}
}

func TestNormalizePhoneBrazilianInputs(t *testing.T) {
	for _, input := range []string{
		"(27) 99999-9999",
		"27 99999-9999",
		"+55 27 99999-9999",
	} {
		got, err := NormalizePhone(input)
		if err != nil {
			t.Fatalf("expected valid phone %q, got %v", input, err)
		}
		if got != "+5527999999999" {
			t.Fatalf("expected canonical phone for %q, got %q", input, got)
		}
	}
}

func TestNormalizePhoneRejectsInvalidFormats(t *testing.T) {
	for _, input := range []string{
		"27 9999",
		"+54 27 99999-9999",
		"02 99999-9999",
		"27 89999-9999",
		"telefone",
	} {
		if _, err := NormalizePhone(input); err == nil {
			t.Fatalf("expected invalid phone %q to fail", input)
		}
	}
}

func TestNormalizePostalCode(t *testing.T) {
	for _, input := range []string{"29100-000", "29100000"} {
		got, err := NormalizePostalCode(input)
		if err != nil {
			t.Fatalf("expected valid postal code %q, got %v", input, err)
		}
		if got != "29100000" {
			t.Fatalf("expected normalized postal code, got %q", got)
		}
	}
}

func TestNormalizePostalCodeRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"2910-000", "29100-0000", "2910A000"} {
		if _, err := NormalizePostalCode(input); err == nil {
			t.Fatalf("expected invalid postal code %q to fail", input)
		}
	}
}

func TestNormalizeState(t *testing.T) {
	for input, want := range map[string]string{
		"ES": "ES",
		"es": "ES",
		"SP": "SP",
		"DF": "DF",
	} {
		got, err := NormalizeState(input)
		if err != nil {
			t.Fatalf("expected valid UF %q, got %v", input, err)
		}
		if got != want {
			t.Fatalf("expected UF %q, got %q", want, got)
		}
	}
}

func TestNormalizeStateRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"XX", "ZZ", "E", "ESP"} {
		if _, err := NormalizeState(input); err == nil {
			t.Fatalf("expected invalid UF %q to fail", input)
		}
	}
}

func TestNormalizeCheckoutInput(t *testing.T) {
	details, values, fieldErrors := NormalizeCheckoutInput(CheckoutInput{
		FullName:    " Joao Silva ",
		Email:       " JOAO@example.COM ",
		Phone:       "(27) 99999-9999",
		CPF:         "529.982.247-25",
		PostalCode:  "29100-000",
		Street:      " Rua Um ",
		Number:      "12A",
		Complement:  " Apto 302 ",
		District:    " Centro ",
		City:        " Vila Velha ",
		State:       "es",
		CountryCode: "",
	})
	if fieldErrors.Any() {
		t.Fatalf("expected no field errors, got %#v", fieldErrors)
	}

	if details.Customer.Email != "joao@example.com" || values.Email != "joao@example.com" {
		t.Fatalf("expected normalized email, got details=%q values=%q", details.Customer.Email, values.Email)
	}
	if details.Customer.Phone != "+5527999999999" || details.Customer.CPF != "52998224725" {
		t.Fatalf("expected normalized phone and CPF, got %#v", details.Customer)
	}
	if details.Address.PostalCode != "29100000" || details.Address.State != "ES" || details.Address.CountryCode != CountryCodeBR {
		t.Fatalf("expected normalized address, got %#v", details.Address)
	}
	if details.Address.Complement == nil || *details.Address.Complement != "Apto 302" {
		t.Fatalf("expected normalized complement, got %#v", details.Address.Complement)
	}
}

func TestNormalizeCheckoutInputRejectsNonBR(t *testing.T) {
	_, _, fieldErrors := NormalizeCheckoutInput(validCheckoutInput(func(input *CheckoutInput) {
		input.CountryCode = "US"
	}))
	if fieldErrors.Message("country_code") == "" {
		t.Fatal("expected country_code error")
	}
}

func validCheckoutInput(modify func(*CheckoutInput)) CheckoutInput {
	input := CheckoutInput{
		FullName:    "Joao Silva",
		Email:       "joao@example.com",
		Phone:       "(27) 99999-9999",
		CPF:         "529.982.247-25",
		PostalCode:  "29100-000",
		Street:      "Rua Um",
		Number:      "12A",
		District:    "Centro",
		City:        "Vila Velha",
		State:       "ES",
		CountryCode: CountryCodeBR,
	}
	if modify != nil {
		modify(&input)
	}

	return input
}
