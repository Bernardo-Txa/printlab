package customers

import (
	"net/mail"
	"strings"
	"unicode"
)

func NormalizeCheckoutInput(input CheckoutInput) (CheckoutDetails, CheckoutInput, FieldErrors) {
	values := sanitizeInput(input)
	errorsByField := FieldErrors{}

	fullName, err := normalizeRequiredText(values.FullName, MaxFullNameLength)
	if err != nil {
		errorsByField["full_name"] = "Informe seu nome."
	} else {
		values.FullName = fullName
	}

	email, err := NormalizeEmail(values.Email)
	if err != nil {
		errorsByField["email"] = "Informe um e-mail valido."
	} else {
		values.Email = email
	}

	phone, err := NormalizePhone(values.Phone)
	if err != nil {
		errorsByField["phone"] = "Informe um telefone brasileiro valido."
	} else {
		values.Phone = phone
	}

	cpf, err := NormalizeCPF(values.CPF)
	if err != nil {
		errorsByField["cpf"] = "Informe um CPF valido."
	} else {
		values.CPF = cpf
	}

	postalCode, err := NormalizePostalCode(values.PostalCode)
	if err != nil {
		errorsByField["postal_code"] = "Informe um CEP valido."
	} else {
		values.PostalCode = postalCode
	}

	street, err := normalizeRequiredText(values.Street, MaxStreetLength)
	if err != nil {
		errorsByField["street"] = "Informe o logradouro."
	} else {
		values.Street = street
	}

	number, err := normalizeRequiredText(values.Number, MaxNumberLength)
	if err != nil {
		errorsByField["number"] = "Informe o numero."
	} else {
		values.Number = number
	}

	complement := normalizeOptionalText(values.Complement, MaxComplementLength)
	if values.Complement != "" && complement == nil {
		errorsByField["complement"] = "Revise o complemento."
	} else if complement != nil {
		values.Complement = *complement
	} else {
		values.Complement = ""
	}

	district, err := normalizeRequiredText(values.District, MaxDistrictLength)
	if err != nil {
		errorsByField["district"] = "Informe o bairro."
	} else {
		values.District = district
	}

	city, err := normalizeRequiredText(values.City, MaxCityLength)
	if err != nil {
		errorsByField["city"] = "Informe a cidade."
	} else {
		values.City = city
	}

	state, err := NormalizeState(values.State)
	if err != nil {
		errorsByField["state"] = "Informe uma UF brasileira valida."
	} else {
		values.State = state
	}

	countryCode, err := NormalizeCountryCode(values.CountryCode)
	if err != nil {
		errorsByField["country_code"] = "Pais indisponivel nesta etapa."
	} else {
		values.CountryCode = countryCode
	}

	if errorsByField.Any() {
		return CheckoutDetails{}, values, errorsByField
	}

	return CheckoutDetails{
		Customer: CustomerDetails{
			FullName: fullName,
			Email:    email,
			Phone:    phone,
			CPF:      cpf,
		},
		Address: ShippingAddress{
			PostalCode:  postalCode,
			Street:      street,
			Number:      number,
			Complement:  complement,
			District:    district,
			City:        city,
			State:       state,
			CountryCode: countryCode,
		},
	}, values, nil
}

func InputFromDetails(details CheckoutDetails) CheckoutInput {
	input := CheckoutInput{
		FullName:    details.Customer.FullName,
		Email:       details.Customer.Email,
		Phone:       details.Customer.Phone,
		CPF:         details.Customer.CPF,
		PostalCode:  details.Address.PostalCode,
		Street:      details.Address.Street,
		Number:      details.Address.Number,
		District:    details.Address.District,
		City:        details.Address.City,
		State:       details.Address.State,
		CountryCode: details.Address.CountryCode,
	}
	if details.Address.Complement != nil {
		input.Complement = *details.Address.Complement
	}
	if input.CountryCode == "" {
		input.CountryCode = CountryCodeBR
	}

	return input
}

func NormalizeCPF(value string) (string, error) {
	digits, err := maskedDigits(value, CPFLength, ".-")
	if err != nil {
		return "", ErrInvalidDetails
	}

	if repeatedDigits(digits) || !validCPFCheckDigits(digits) {
		return "", ErrInvalidDetails
	}

	return digits, nil
}

func NormalizeEmail(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || len(normalized) > MaxEmailLength || strings.ContainsAny(normalized, "\r\n") {
		return "", ErrInvalidDetails
	}

	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized {
		return "", ErrInvalidDetails
	}

	parts := strings.Split(normalized, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ErrInvalidDetails
	}

	return normalized, nil
}

func NormalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidDetails
	}

	hasPlus := false
	var digits strings.Builder
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
			digits.WriteRune(char)
		case char == '+' && digits.Len() == 0 && !hasPlus:
			hasPlus = true
		case char == '(' || char == ')' || char == '-' || char == '.' || unicode.IsSpace(char):
			continue
		default:
			return "", ErrInvalidDetails
		}
	}

	national := digits.String()
	if hasPlus {
		if !strings.HasPrefix(national, "55") {
			return "", ErrInvalidDetails
		}
		national = strings.TrimPrefix(national, "55")
	} else if strings.HasPrefix(national, "55") && (len(national) == 12 || len(national) == 13) {
		national = strings.TrimPrefix(national, "55")
	}

	if len(national) != 10 && len(national) != 11 {
		return "", ErrInvalidDetails
	}
	if !validBrazilianDDD(national[:2]) {
		return "", ErrInvalidDetails
	}
	if len(national) == 11 && national[2] != '9' {
		return "", ErrInvalidDetails
	}

	normalized := "+55" + national
	if len(normalized) > MaxPhoneLength {
		return "", ErrInvalidDetails
	}

	return normalized, nil
}

func NormalizePostalCode(value string) (string, error) {
	return maskedDigits(value, PostalCodeLength, "-")
}

func NormalizeState(value string) (string, error) {
	state := strings.ToUpper(strings.TrimSpace(value))
	if !validBrazilianState(state) {
		return "", ErrInvalidDetails
	}

	return state, nil
}

func NormalizeCountryCode(value string) (string, error) {
	countryCode := strings.ToUpper(strings.TrimSpace(value))
	if countryCode == "" {
		countryCode = CountryCodeBR
	}
	if countryCode != CountryCodeBR {
		return "", ErrInvalidDetails
	}

	return countryCode, nil
}

func sanitizeInput(input CheckoutInput) CheckoutInput {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.CPF = strings.TrimSpace(input.CPF)
	input.PostalCode = strings.TrimSpace(input.PostalCode)
	input.Street = strings.TrimSpace(input.Street)
	input.Number = strings.TrimSpace(input.Number)
	input.Complement = strings.TrimSpace(input.Complement)
	input.District = strings.TrimSpace(input.District)
	input.City = strings.TrimSpace(input.City)
	input.State = strings.ToUpper(strings.TrimSpace(input.State))
	input.CountryCode = strings.ToUpper(strings.TrimSpace(input.CountryCode))
	if input.CountryCode == "" {
		input.CountryCode = CountryCodeBR
	}

	return input
}

func normalizeRequiredText(value string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > maxLength {
		return "", ErrInvalidDetails
	}

	return value, nil
}

func normalizeOptionalText(value string, maxLength int) *string {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > maxLength {
		return nil
	}

	return &value
}

func maskedDigits(value string, length int, allowedMask string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidDetails
	}

	var digits strings.Builder
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
			digits.WriteRune(char)
		case unicode.IsSpace(char) || strings.ContainsRune(allowedMask, char):
			continue
		default:
			return "", ErrInvalidDetails
		}
	}

	if digits.Len() != length {
		return "", ErrInvalidDetails
	}

	return digits.String(), nil
}

func repeatedDigits(value string) bool {
	if value == "" {
		return false
	}

	for i := 1; i < len(value); i++ {
		if value[i] != value[0] {
			return false
		}
	}

	return true
}

func validCPFCheckDigits(cpf string) bool {
	first := cpfDigit(cpf[:9], 10)
	firstDigit := byte('0' + first)
	second := cpfDigit(cpf[:9]+string(firstDigit), 11)
	secondDigit := byte('0' + second)

	return cpf[9] == firstDigit && cpf[10] == secondDigit
}

func cpfDigit(base string, weight int) int {
	sum := 0
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * (weight - i)
	}

	rest := (sum * 10) % 11
	if rest == 10 {
		return 0
	}

	return rest
}

func validBrazilianDDD(ddd string) bool {
	_, ok := brazilianDDDs[ddd]
	return ok
}

var brazilianDDDs = map[string]struct{}{
	"11": {}, "12": {}, "13": {}, "14": {}, "15": {}, "16": {}, "17": {}, "18": {}, "19": {},
	"21": {}, "22": {}, "24": {}, "27": {}, "28": {},
	"31": {}, "32": {}, "33": {}, "34": {}, "35": {}, "37": {}, "38": {},
	"41": {}, "42": {}, "43": {}, "44": {}, "45": {}, "46": {}, "47": {}, "48": {}, "49": {},
	"51": {}, "53": {}, "54": {}, "55": {},
	"61": {}, "62": {}, "63": {}, "64": {}, "65": {}, "66": {}, "67": {}, "68": {}, "69": {},
	"71": {}, "73": {}, "74": {}, "75": {}, "77": {}, "79": {},
	"81": {}, "82": {}, "83": {}, "84": {}, "85": {}, "86": {}, "87": {}, "88": {}, "89": {},
	"91": {}, "92": {}, "93": {}, "94": {}, "95": {}, "96": {}, "97": {}, "98": {}, "99": {},
}

func validBrazilianState(state string) bool {
	_, ok := brazilianStates[state]
	return ok
}

var brazilianStates = map[string]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {}, "ES": {}, "GO": {},
	"MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {}, "PB": {}, "PR": {}, "PE": {}, "PI": {},
	"RJ": {}, "RN": {}, "RS": {}, "RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {},
}
