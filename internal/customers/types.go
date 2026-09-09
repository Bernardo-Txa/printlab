package customers

import "errors"

const (
	MaxFullNameLength   = 120
	MaxEmailLength      = 254
	MaxPhoneLength      = 20
	CPFLength           = 11
	PostalCodeLength    = 8
	MaxStreetLength     = 160
	MaxNumberLength     = 30
	MaxComplementLength = 120
	MaxDistrictLength   = 100
	MaxCityLength       = 100
	CountryCodeBR       = "BR"
)

var (
	ErrInvalidDetails   = errors.New("invalid customer details")
	ErrCartRequired     = errors.New("cart required")
	ErrEmptyCart        = errors.New("empty cart")
	ErrUnavailableItems = errors.New("cart has unavailable items")
	ErrNotFound         = errors.New("customer details not found")
	ErrUnavailable      = errors.New("customer details unavailable")
)

type CustomerDetails struct {
	FullName string
	Email    string
	Phone    string
	CPF      string
}

type ShippingAddress struct {
	PostalCode  string
	Street      string
	Number      string
	Complement  *string
	District    string
	City        string
	State       string
	CountryCode string
}

type CheckoutDetails struct {
	Customer CustomerDetails
	Address  ShippingAddress
}

type CheckoutInput struct {
	FullName    string
	Email       string
	Phone       string
	CPF         string
	PostalCode  string
	Street      string
	Number      string
	Complement  string
	District    string
	City        string
	State       string
	CountryCode string
}

type FieldErrors map[string]string

func (e FieldErrors) Any() bool {
	return len(e) > 0
}

func (e FieldErrors) Message(field string) string {
	if e == nil {
		return ""
	}

	return e[field]
}

type CheckoutForm struct {
	Values CheckoutInput
	Errors FieldErrors
	Saved  bool
}

func (f CheckoutForm) HasErrors() bool {
	return f.Errors.Any()
}
