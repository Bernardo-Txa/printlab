package config

import (
	"errors"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPort       = "8080"
	DefaultDBMaxConns = int32(4)
)

var (
	ErrInvalidDatabaseURL = errors.New("invalid DATABASE_URL")
	ErrInvalidDBMaxConns  = errors.New("invalid DB_MAX_CONNS")
)

type Config struct {
	AppEnv             string
	VercelEnv          string
	Port               string
	SiteURL            string
	DatabaseURL        string
	DatabaseConfigured bool
	DBMaxConns         int32
	SupabaseURL        string
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

	return Config{
		AppEnv:             strings.TrimSpace(value(lookup, "APP_ENV")),
		VercelEnv:          strings.TrimSpace(value(lookup, "VERCEL_ENV")),
		Port:               port,
		SiteURL:            strings.TrimSpace(value(lookup, "SITE_URL")),
		DatabaseURL:        databaseURL,
		DatabaseConfigured: databaseConfigured,
		DBMaxConns:         dbMaxConns,
		SupabaseURL:        strings.TrimRight(strings.TrimSpace(value(lookup, "SUPABASE_URL")), "/"),
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
