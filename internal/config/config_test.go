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

func mapLookup(values map[string]string) envLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
