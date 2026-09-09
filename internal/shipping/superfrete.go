package shipping

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	SuperFreteSandboxBaseURL    = "https://sandbox.superfrete.com"
	SuperFreteProductionBaseURL = "https://api.superfrete.com"
	superFreteCalculatorPath    = "/api/v0/calculator"
	defaultHTTPTimeout          = 8 * time.Second
)

type SuperFreteClientError struct {
	Category   string
	StatusCode int
}

func (e *SuperFreteClientError) Error() string {
	if e == nil {
		return ErrUnavailable.Error()
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: superfrete category=%s status=%d", ErrUnavailable, e.Category, e.StatusCode)
	}

	return fmt.Sprintf("%s: superfrete category=%s", ErrUnavailable, e.Category)
}

func (e *SuperFreteClientError) Unwrap() error {
	return ErrUnavailable
}

type SuperFreteClientConfig struct {
	Environment  string
	APIToken     string
	ContactEmail string
	HTTPClient   *http.Client
	BaseURL      string
}

type SuperFreteClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	userAgent  string
}

type SuperFreteCalculatorRequest struct {
	FromPostalCode string
	ToPostalCode   string
	Services       string
	Products       []SuperFreteProduct
	Package        *SuperFretePackage
}

type SuperFreteProduct struct {
	Quantity int
	WeightKG float64
	HeightCM float64
	WidthCM  float64
	LengthCM float64
}

type SuperFretePackage struct {
	WeightKG float64
	HeightCM float64
	WidthCM  float64
	LengthCM float64
}

type SuperFreteQuote struct {
	ServiceCode      string
	ServiceName      string
	CarrierName      string
	PriceCents       int64
	DeliveryTimeDays *int
	Package          *SuperFreteReturnedPackage
}

type SuperFreteReturnedPackage struct {
	WeightKG float64
	HeightMM int
	WidthMM  int
	LengthMM int
}

func NewSuperFreteClient(cfg SuperFreteClientConfig) (*SuperFreteClient, error) {
	baseURL, err := superFreteBaseURL(cfg.Environment, cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = defaultHTTPTimeout
	}

	token := strings.TrimSpace(cfg.APIToken)
	if token == "" {
		return nil, ErrNotConfigured
	}

	contactEmail := strings.TrimSpace(cfg.ContactEmail)
	if contactEmail == "" || strings.ContainsAny(contactEmail, "\r\n") {
		return nil, ErrNotConfigured
	}

	return &SuperFreteClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		token:      token,
		userAgent:  "PrintLab/1.0 (" + contactEmail + ")",
	}, nil
}

func (c *SuperFreteClient) Calculate(ctx context.Context, request SuperFreteCalculatorRequest) ([]SuperFreteQuote, error) {
	if c == nil || c.httpClient == nil || c.baseURL == "" || c.token == "" || c.userAgent == "" {
		return nil, ErrNotConfigured
	}

	payload, err := json.Marshal(toSuperFretePayload(request))
	if err != nil {
		return nil, ErrUnavailable
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+superFreteCalculatorPath, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrUnavailable
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.token)
	httpRequest.Header.Set("User-Agent", c.userAgent)
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		if isSuperFreteTimeout(err) {
			return nil, safeSuperFreteError("timeout", 0)
		}

		return nil, safeSuperFreteError("request_failed", 0)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, safeSuperFreteError("response_read_failed", 0)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, safeSuperFreteError(superFreteStatusCategory(response.StatusCode), response.StatusCode)
	}

	var externalQuotes []superFreteQuoteDTO
	if err := json.Unmarshal(body, &externalQuotes); err != nil {
		return nil, safeSuperFreteError("invalid_json", 0)
	}

	return mapSuperFreteQuotes(externalQuotes), nil
}

func superFreteBaseURL(environment string, testBaseURL string) (string, error) {
	if testBaseURL != "" {
		parsed, err := url.Parse(testBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return "", ErrNotConfigured
		}

		return strings.TrimRight(testBaseURL, "/"), nil
	}

	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "sandbox":
		return SuperFreteSandboxBaseURL, nil
	case "production":
		return SuperFreteProductionBaseURL, nil
	default:
		return "", ErrNotConfigured
	}
}

func safeSuperFreteError(category string, statusCode int) error {
	return &SuperFreteClientError{Category: category, StatusCode: statusCode}
}

func superFreteStatusCategory(statusCode int) string {
	switch {
	case statusCode == http.StatusBadRequest:
		return "400"
	case statusCode == http.StatusUnauthorized:
		return "401"
	case statusCode == http.StatusTooManyRequests:
		return "429"
	case statusCode >= http.StatusInternalServerError:
		return "500"
	default:
		return "http_status"
	}
}

func isSuperFreteTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

type superFretePayload struct {
	From     superFretePostalCodeDTO `json:"from"`
	To       superFretePostalCodeDTO `json:"to"`
	Services string                  `json:"services"`
	Options  superFreteOptionsDTO    `json:"options"`
	Products []superFreteProductDTO  `json:"products,omitempty"`
	Package  *superFretePackageDTO   `json:"package,omitempty"`
}

type superFretePostalCodeDTO struct {
	PostalCode string `json:"postal_code"`
}

type superFreteOptionsDTO struct {
	OwnHand           bool    `json:"own_hand"`
	Receipt           bool    `json:"receipt"`
	InsuranceValue    float64 `json:"insurance_value"`
	UseInsuranceValue bool    `json:"use_insurance_value"`
}

type superFreteProductDTO struct {
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight"`
	Height   float64 `json:"height"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
}

type superFretePackageDTO struct {
	Weight float64 `json:"weight"`
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
	Length float64 `json:"length"`
}

func toSuperFretePayload(request SuperFreteCalculatorRequest) superFretePayload {
	payload := superFretePayload{
		From:     superFretePostalCodeDTO{PostalCode: request.FromPostalCode},
		To:       superFretePostalCodeDTO{PostalCode: request.ToPostalCode},
		Services: request.Services,
		Options: superFreteOptionsDTO{
			OwnHand:           false,
			Receipt:           false,
			InsuranceValue:    0,
			UseInsuranceValue: false,
		},
	}

	if len(request.Products) > 0 {
		payload.Products = make([]superFreteProductDTO, 0, len(request.Products))
		for _, product := range request.Products {
			payload.Products = append(payload.Products, superFreteProductDTO{
				Quantity: product.Quantity,
				Weight:   product.WeightKG,
				Height:   product.HeightCM,
				Width:    product.WidthCM,
				Length:   product.LengthCM,
			})
		}
	}

	if request.Package != nil {
		payload.Package = &superFretePackageDTO{
			Weight: request.Package.WeightKG,
			Height: request.Package.HeightCM,
			Width:  request.Package.WidthCM,
			Length: request.Package.LengthCM,
		}
	}

	return payload
}

type superFreteQuoteDTO struct {
	ID           int                   `json:"id"`
	Name         string                `json:"name"`
	Price        json.RawMessage       `json:"price"`
	DeliveryTime *int                  `json:"delivery_time"`
	Packages     []superFretePackageIn `json:"packages"`
	Company      superFreteCompanyDTO  `json:"company"`
	HasError     bool                  `json:"has_error"`
	Error        string                `json:"error"`
}

type superFretePackageIn struct {
	Price      json.RawMessage             `json:"price"`
	Dimensions superFreteDimensionsDTO     `json:"dimensions"`
	Weight     flexibleSuperFreteDimension `json:"weight"`
}

type superFreteDimensionsDTO struct {
	Height flexibleSuperFreteDimension `json:"height"`
	Width  flexibleSuperFreteDimension `json:"width"`
	Length flexibleSuperFreteDimension `json:"length"`
}

type superFreteCompanyDTO struct {
	Name string `json:"name"`
}

type flexibleSuperFreteDimension struct {
	value string
}

func (d *flexibleSuperFreteDimension) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return errors.New("empty superfrete dimension")
	}
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		d.value = value
		return nil
	}

	d.value = string(data)
	return nil
}

func mapSuperFreteQuotes(externalQuotes []superFreteQuoteDTO) []SuperFreteQuote {
	quotes := make([]SuperFreteQuote, 0, len(externalQuotes))
	for _, external := range externalQuotes {
		if external.HasError {
			logSuperFreteServiceError(external)
			continue
		}
		if external.ID == 0 || strings.TrimSpace(external.Name) == "" {
			continue
		}

		priceCents, err := DecimalToCents(string(external.Price))
		if err != nil {
			continue
		}

		quote := SuperFreteQuote{
			ServiceCode:      strconvItoa(external.ID),
			ServiceName:      strings.TrimSpace(external.Name),
			CarrierName:      strings.TrimSpace(external.Company.Name),
			PriceCents:       priceCents,
			DeliveryTimeDays: external.DeliveryTime,
		}

		if len(external.Packages) > 0 {
			returnedPackage, err := mapSuperFretePackage(external.Packages[0])
			if err == nil {
				quote.Package = &returnedPackage
			}
		}

		quotes = append(quotes, quote)
	}

	return quotes
}

func logSuperFreteServiceError(external superFreteQuoteDTO) {
	serviceName := safeSuperFreteLogValue(external.Name)
	switch {
	case external.ID != 0 && serviceName != "":
		log.Printf("shipping quote service unavailable service_code=%d service_name=%q reason=service_error", external.ID, serviceName)
	case external.ID != 0:
		log.Printf("shipping quote service unavailable service_code=%d reason=service_error", external.ID)
	case serviceName != "":
		log.Printf("shipping quote service unavailable service_name=%q reason=service_error", serviceName)
	default:
		log.Print("shipping quote service unavailable reason=service_error")
	}
}

func safeSuperFreteLogValue(value string) string {
	value = strings.TrimSpace(strings.Map(func(char rune) rune {
		switch char {
		case '\r', '\n', '\t':
			return -1
		default:
			return char
		}
	}, value))
	runes := []rune(value)
	if len(runes) > 80 {
		return string(runes[:80])
	}

	return value
}

func mapSuperFretePackage(external superFretePackageIn) (SuperFreteReturnedPackage, error) {
	heightMM, err := CentimetersToMillimetersCeil(external.Dimensions.Height.value)
	if err != nil {
		return SuperFreteReturnedPackage{}, err
	}
	widthMM, err := CentimetersToMillimetersCeil(external.Dimensions.Width.value)
	if err != nil {
		return SuperFreteReturnedPackage{}, err
	}
	lengthMM, err := CentimetersToMillimetersCeil(external.Dimensions.Length.value)
	if err != nil {
		return SuperFreteReturnedPackage{}, err
	}

	weight, err := decimalToScaledRounded(external.Weight.value, 1000)
	if err != nil {
		return SuperFreteReturnedPackage{}, err
	}

	return SuperFreteReturnedPackage{
		WeightKG: float64(weight) / 1000,
		HeightMM: heightMM,
		WidthMM:  widthMM,
		LengthMM: lengthMM,
	}, nil
}

func strconvItoa(value int) string {
	return fmt.Sprintf("%d", value)
}
