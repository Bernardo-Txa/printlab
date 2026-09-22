package customerauth

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	MaxNameLength     = 120
	MaxEmailLength    = 254
	MinPasswordLength = 8
	MaxPasswordLength = 1024
)

func ValidateSignup(name string, email string, password string, confirmPassword string) (SignUpInput, FieldErrors) {
	input := SignUpInput{
		Name:     strings.TrimSpace(name),
		Email:    normalizeEmail(email),
		Password: password,
	}
	errors := FieldErrors{}
	if input.Name == "" {
		errors["name"] = "Informe seu nome."
	} else if utf8.RuneCountInString(input.Name) > MaxNameLength {
		errors["name"] = "Informe um nome mais curto."
	}
	if !validEmail(input.Email) {
		errors["email"] = "Informe um e-mail válido."
	}
	validatePassword(password, errors)
	if strings.TrimSpace(confirmPassword) == "" {
		errors["confirm_password"] = "Confirme sua senha."
	} else if password != confirmPassword {
		errors["confirm_password"] = "As senhas não coincidem."
	}
	return input, errors
}

func ValidateLogin(email string, password string, next string) (string, string, string, FieldErrors) {
	errors := FieldErrors{}
	normalizedEmail := normalizeEmail(email)
	if !validEmail(normalizedEmail) {
		errors["email"] = "Informe um e-mail válido."
	}
	if strings.TrimSpace(password) == "" {
		errors["password"] = "Informe sua senha."
	}
	safeNext, err := SafeRedirectPath(next, "/conta")
	if err != nil {
		safeNext = "/conta"
	}
	return normalizedEmail, password, safeNext, errors
}

func ValidateRecovery(email string) (string, FieldErrors) {
	normalizedEmail := normalizeEmail(email)
	errors := FieldErrors{}
	if !validEmail(normalizedEmail) {
		errors["email"] = "Informe um e-mail válido."
	}
	return normalizedEmail, errors
}

func ValidateNewPassword(password string, confirmPassword string) FieldErrors {
	errors := FieldErrors{}
	validatePassword(password, errors)
	if strings.TrimSpace(confirmPassword) == "" {
		errors["confirm_password"] = "Confirme a nova senha."
	} else if password != confirmPassword {
		errors["confirm_password"] = "As senhas não coincidem."
	}
	return errors
}

func validatePassword(password string, errors FieldErrors) {
	if strings.TrimSpace(password) == "" {
		errors["password"] = "Informe sua senha."
		return
	}
	length := utf8.RuneCountInString(password)
	if length < MinPasswordLength {
		errors["password"] = "Use pelo menos 8 caracteres."
	} else if length > MaxPasswordLength {
		errors["password"] = "Use uma senha mais curta."
	}
}

func validEmail(email string) bool {
	if email == "" || len(email) > MaxEmailLength || strings.ContainsAny(email, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func SafeRedirectPath(value string, fallback string) (string, error) {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" || !strings.HasPrefix(fallback, "/") || strings.HasPrefix(fallback, "//") {
		fallback = "/conta"
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.User != nil || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\r\n") {
		return "", ErrInvalidRedirect
	}
	if strings.HasPrefix(parsed.Path, "/admin") || strings.HasPrefix(parsed.Path, "/auth/session") || strings.HasPrefix(parsed.Path, "/webhooks/") {
		return "", ErrInvalidRedirect
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}
