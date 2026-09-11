package config

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadDefaultDBMaxConnsWhenAbsent(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DBMaxConns != DefaultDBMaxConns {
		t.Fatalf("expected DB max conns %d, got %d", DefaultDBMaxConns, cfg.DBMaxConns)
	}
}

func TestLoadDBMaxConnsFromEnv(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{"DB_MAX_CONNS": "8"}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DBMaxConns != 8 {
		t.Fatalf("expected DB max conns 8, got %d", cfg.DBMaxConns)
	}
}

func TestLoadRejectsInvalidDBMaxConns(t *testing.T) {
	for _, value := range []string{"abc", "0", "-1"} {
		t.Run(value, func(t *testing.T) {
			_, err := loadFromEnv(mapLookup(map[string]string{"DB_MAX_CONNS": value}))
			if !errors.Is(err, ErrInvalidDBMaxConns) {
				t.Fatalf("expected ErrInvalidDBMaxConns, got %v", err)
			}
		})
	}
}

func TestLoadAllowsMissingDatabaseURL(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DatabaseConfigured {
		t.Fatal("expected database to be marked as not configured")
	}

	if cfg.DatabaseURL != "" {
		t.Fatal("expected empty database URL")
	}
}

func TestLoadRejectsInvalidDatabaseURLSafely(t *testing.T) {
	_, err := loadFromEnv(mapLookup(map[string]string{
		"DATABASE_URL": "postgresql://user@%zz",
	}))
	if !errors.Is(err, ErrInvalidDatabaseURL) {
		t.Fatalf("expected ErrInvalidDatabaseURL, got %v", err)
	}

	if strings.Contains(err.Error(), "postgresql://") {
		t.Fatal("expected error not to expose database password")
	}
}

func TestLoadAcceptsPostgresDatabaseURL(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{
		"DATABASE_URL": "postgresql://user@example.com:5432/postgres",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cfg.DatabaseConfigured {
		t.Fatal("expected database to be marked as configured")
	}
}

func TestLoadTrimsOptionalSupabaseURL(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{
		"SUPABASE_URL": " https://example.supabase.co/ ",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.SupabaseURL != "https://example.supabase.co" {
		t.Fatalf("expected trimmed Supabase URL, got %q", cfg.SupabaseURL)
	}
}

func TestLoadReadsOptionalSiteAndRuntimeEnvironment(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{
		"APP_ENV":    " production ",
		"VERCEL_ENV": " preview ",
		"SITE_URL":   " https://printlab.example ",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppEnv != "production" || cfg.VercelEnv != "preview" || cfg.SiteURL != "https://printlab.example" {
		t.Fatalf("expected optional environment values to be trimmed, got %#v", cfg)
	}
}

func TestLoadAllowsMissingSuperFreteConfig(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.SuperFreteConfigured {
		t.Fatal("expected SuperFrete to be unconfigured")
	}
}

func TestLoadAllowsMissingInfinitePayHandle(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.InfinitePayConfigured {
		t.Fatal("expected InfinitePay to be unconfigured")
	}
	if cfg.InfinitePayHandle != "" {
		t.Fatalf("expected empty InfinitePay handle, got %q", cfg.InfinitePayHandle)
	}
}

func TestLoadReadsInfinitePayHandle(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{
		"INFINITEPAY_HANDLE": " printlab-handle_01 ",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cfg.InfinitePayConfigured {
		t.Fatal("expected InfinitePay to be configured")
	}
	if cfg.InfinitePayHandle != "printlab-handle_01" {
		t.Fatalf("expected trimmed InfinitePay handle, got %q", cfg.InfinitePayHandle)
	}
}

func TestLoadRejectsInvalidInfinitePayHandle(t *testing.T) {
	for _, value := range []string{"$printlab", "print lab", "printlab/checkout", strings.Repeat("a", 101)} {
		t.Run(value, func(t *testing.T) {
			_, err := loadFromEnv(mapLookup(map[string]string{"INFINITEPAY_HANDLE": value}))
			if !errors.Is(err, ErrInvalidInfinitePayHandle) {
				t.Fatalf("expected ErrInvalidInfinitePayHandle, got %v", err)
			}
		})
	}
}

func TestLoadReadsSuperFreteConfig(t *testing.T) {
	cfg, err := loadFromEnv(mapLookup(map[string]string{
		"SUPERFRETE_ENV":                " sandbox ",
		"SUPERFRETE_API_TOKEN":          " token-value ",
		"SUPERFRETE_ORIGIN_POSTAL_CODE": "01153-000",
		"SUPERFRETE_CONTACT_EMAIL":      " INTEGRACAO@example.com ",
		"SUPERFRETE_SERVICES":           "1, 2,17,3,33",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cfg.SuperFreteConfigured {
		t.Fatal("expected SuperFrete to be configured")
	}
	if cfg.SuperFreteEnv != "sandbox" || cfg.SuperFreteOriginPostalCode != "01153000" {
		t.Fatalf("expected normalized SuperFrete env and origin, got %#v", cfg)
	}
	if cfg.SuperFreteContactEmail != "integracao@example.com" {
		t.Fatalf("expected normalized contact email, got %q", cfg.SuperFreteContactEmail)
	}
	if strings.Join(cfg.SuperFreteServiceCodes, ",") != "1,2,17,3,33" || cfg.SuperFreteServiceCodesValue != "1,2,17,3,33" {
		t.Fatalf("expected normalized SuperFrete services, got %#v", cfg.SuperFreteServiceCodes)
	}
}

func TestLoadRejectsIncompleteSuperFreteConfig(t *testing.T) {
	_, err := loadFromEnv(mapLookup(map[string]string{
		"SUPERFRETE_ENV": "sandbox",
	}))
	if !errors.Is(err, ErrIncompleteSuperFreteConfig) {
		t.Fatalf("expected ErrIncompleteSuperFreteConfig, got %v", err)
	}
}

func TestLoadRejectsInvalidSuperFreteConfig(t *testing.T) {
	base := map[string]string{
		"SUPERFRETE_ENV":                "sandbox",
		"SUPERFRETE_API_TOKEN":          "token-value",
		"SUPERFRETE_ORIGIN_POSTAL_CODE": "01153000",
		"SUPERFRETE_CONTACT_EMAIL":      "integracao@example.com",
		"SUPERFRETE_SERVICES":           "1,2",
	}

	tests := []struct {
		name string
		key  string
		val  string
		want error
	}{
		{name: "env", key: "SUPERFRETE_ENV", val: "staging", want: ErrInvalidSuperFreteEnv},
		{name: "origin postal code", key: "SUPERFRETE_ORIGIN_POSTAL_CODE", val: "0115A000", want: ErrInvalidSuperFreteOriginPostalCode},
		{name: "contact email", key: "SUPERFRETE_CONTACT_EMAIL", val: "not-an-email", want: ErrInvalidSuperFreteContactEmail},
		{name: "unknown service", key: "SUPERFRETE_SERVICES", val: "1,31", want: ErrInvalidSuperFreteServices},
		{name: "duplicate service", key: "SUPERFRETE_SERVICES", val: "1,1", want: ErrInvalidSuperFreteServices},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{}
			for key, value := range base {
				values[key] = value
			}
			values[tt.key] = tt.val

			_, err := loadFromEnv(mapLookup(values))
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func mapLookup(values map[string]string) envLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
