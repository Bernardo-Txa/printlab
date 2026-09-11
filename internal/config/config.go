package config

import (
	"errors"
	"math"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultPort       = "8080"
	DefaultDBMaxConns = int32(4)
)

var (
	ErrInvalidDatabaseURL                = errors.New("invalid DATABASE_URL")
	ErrInvalidDBMaxConns                 = errors.New("invalid DB_MAX_CONNS")
	ErrIncompleteSuperFreteConfig        = errors.New("incomplete SuperFrete configuration")
	ErrInvalidSuperFreteEnv              = errors.New("invalid SUPERFRETE_ENV")
	ErrInvalidSuperFreteOriginPostalCode = errors.New("invalid SUPERFRETE_ORIGIN_POSTAL_CODE")
	ErrInvalidSuperFreteContactEmail     = errors.New("invalid SUPERFRETE_CONTACT_EMAIL")
	ErrInvalidSuperFreteServices         = errors.New("invalid SUPERFRETE_SERVICES")
	ErrInvalidInfinitePayHandle          = errors.New("invalid INFINITEPAY_HANDLE")
)

var infinitePayHandlePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

type Config struct {
	AppEnv                      string
	VercelEnv                   string
	Port                        string
	SiteURL                     string
	DatabaseURL                 string
	DatabaseConfigured          bool
	DBMaxConns                  int32
	SupabaseURL                 string
	SuperFreteConfigured        bool
	SuperFreteEnv               string
	SuperFreteAPIToken          string
	SuperFreteOriginPostalCode  string
	SuperFreteContactEmail      string
	SuperFreteServiceCodes      []string
	SuperFreteServiceCodesValue string
	InfinitePayHandle           string
	InfinitePayConfigured       bool
}

type envLookup func(string) (string, bool)

func Load() (Config, error) {
	return loadFromEnv(os.LookupEnv)
}

func loadFromEnv(lookup envLookup) (Config, error) {
	databaseURL := strings.TrimSpace(value(lookup, "DATABASE_URL"))
	databaseConfigured, err := validateDatabaseURL(databaseURL)
	if err != nil {
		return Config{}, err
	}

	dbMaxConns, err := parseDBMaxConns(lookup)
	if err != nil {
		return Config{}, err
	}

	port := strings.TrimSpace(value(lookup, "PORT"))
	if port == "" {
		port = DefaultPort
	}

	superFrete, err := parseSuperFreteConfig(lookup)
	if err != nil {
		return Config{}, err
	}

	infinitePayHandle, err := parseInfinitePayHandle(lookup)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:                      strings.TrimSpace(value(lookup, "APP_ENV")),
		VercelEnv:                   strings.TrimSpace(value(lookup, "VERCEL_ENV")),
		Port:                        port,
		SiteURL:                     strings.TrimSpace(value(lookup, "SITE_URL")),
		DatabaseURL:                 databaseURL,
		DatabaseConfigured:          databaseConfigured,
		DBMaxConns:                  dbMaxConns,
		SupabaseURL:                 strings.TrimRight(strings.TrimSpace(value(lookup, "SUPABASE_URL")), "/"),
		SuperFreteConfigured:        superFrete.configured,
		SuperFreteEnv:               superFrete.environment,
		SuperFreteAPIToken:          superFrete.apiToken,
		SuperFreteOriginPostalCode:  superFrete.originPostalCode,
		SuperFreteContactEmail:      superFrete.contactEmail,
		SuperFreteServiceCodes:      superFrete.serviceCodes,
		SuperFreteServiceCodesValue: strings.Join(superFrete.serviceCodes, ","),
		InfinitePayHandle:           infinitePayHandle,
		InfinitePayConfigured:       infinitePayHandle != "",
	}, nil
}

func value(lookup envLookup, key string) string {
	value, _ := lookup(key)
	return value
}

func parseDBMaxConns(lookup envLookup) (int32, error) {
	raw, ok := lookup("DB_MAX_CONNS")
	if !ok || strings.TrimSpace(raw) == "" {
		return DefaultDBMaxConns, nil
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 || value > math.MaxInt32 {
		return 0, ErrInvalidDBMaxConns
	}

	return int32(value), nil
}

func validateDatabaseURL(databaseURL string) (bool, error) {
	if databaseURL == "" {
		return false, nil
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false, ErrInvalidDatabaseURL
	}

	switch parsed.Scheme {
	case "postgres", "postgresql":
		return true, nil
	default:
		return false, ErrInvalidDatabaseURL
	}
}

type superFreteEnvConfig struct {
	configured       bool
	environment      string
	apiToken         string
	originPostalCode string
	contactEmail     string
	serviceCodes     []string
}

func parseSuperFreteConfig(lookup envLookup) (superFreteEnvConfig, error) {
	environment := strings.ToLower(strings.TrimSpace(value(lookup, "SUPERFRETE_ENV")))
	apiToken := strings.TrimSpace(value(lookup, "SUPERFRETE_API_TOKEN"))
	originPostalCode := strings.TrimSpace(value(lookup, "SUPERFRETE_ORIGIN_POSTAL_CODE"))
	contactEmail := strings.ToLower(strings.TrimSpace(value(lookup, "SUPERFRETE_CONTACT_EMAIL")))
	rawServices := strings.TrimSpace(value(lookup, "SUPERFRETE_SERVICES"))

	if environment == "" && apiToken == "" && originPostalCode == "" && contactEmail == "" && rawServices == "" {
		return superFreteEnvConfig{}, nil
	}

	if environment == "" || apiToken == "" || originPostalCode == "" || contactEmail == "" || rawServices == "" {
		return superFreteEnvConfig{}, ErrIncompleteSuperFreteConfig
	}
	if environment != "sandbox" && environment != "production" {
		return superFreteEnvConfig{}, ErrInvalidSuperFreteEnv
	}

	normalizedPostalCode, ok := normalizeBrazilianPostalCode(originPostalCode)
	if !ok {
		return superFreteEnvConfig{}, ErrInvalidSuperFreteOriginPostalCode
	}

	if !validContactEmail(contactEmail) {
		return superFreteEnvConfig{}, ErrInvalidSuperFreteContactEmail
	}

	serviceCodes, err := normalizeSuperFreteServiceCodes(rawServices)
	if err != nil {
		return superFreteEnvConfig{}, err
	}

	return superFreteEnvConfig{
		configured:       true,
		environment:      environment,
		apiToken:         apiToken,
		originPostalCode: normalizedPostalCode,
		contactEmail:     contactEmail,
		serviceCodes:     serviceCodes,
	}, nil
}

func normalizeBrazilianPostalCode(value string) (string, bool) {
	var digits strings.Builder
	for _, char := range strings.TrimSpace(value) {
		switch {
		case char >= '0' && char <= '9':
			digits.WriteRune(char)
		case char == '-' || strings.ContainsRune(" \t\r\n", char):
			continue
		default:
			return "", false
		}
	}

	postalCode := digits.String()
	return postalCode, len(postalCode) == 8
}

func validContactEmail(value string) bool {
	if value == "" || len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return false
	}

	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func normalizeSuperFreteServiceCodes(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	seen := map[string]bool{}
	var services []string
	for _, part := range parts {
		service := strings.TrimSpace(part)
		if !allowedSuperFreteService(service) || seen[service] {
			return nil, ErrInvalidSuperFreteServices
		}
		seen[service] = true
		services = append(services, service)
	}

	if len(services) == 0 {
		return nil, ErrInvalidSuperFreteServices
	}

	return services, nil
}

func allowedSuperFreteService(service string) bool {
	switch service {
	case "1", "2", "3", "17", "33":
		return true
	default:
		return false
	}
}

func parseInfinitePayHandle(lookup envLookup) (string, error) {
	handle := strings.TrimSpace(value(lookup, "INFINITEPAY_HANDLE"))
	if handle == "" {
		return "", nil
	}
	if strings.HasPrefix(handle, "$") || strings.ContainsAny(handle, " \t\r\n") || !infinitePayHandlePattern.MatchString(handle) {
		return "", ErrInvalidInfinitePayHandle
	}

	return handle, nil
}
