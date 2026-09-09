package database

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestNewWithoutDatabaseURLIsUnconfigured(t *testing.T) {
	db, err := New(context.Background(), Config{MaxConns: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if db.Configured() {
		t.Fatal("expected database to be unconfigured")
	}

	if err := db.Ping(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestNewConfiguresTransactionPoolerCompatibleMode(t *testing.T) {
	db, err := New(context.Background(), Config{
		DatabaseURL: "postgresql://user@example.com:5432/postgres",
		MaxConns:    4,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer db.Close()

	poolConfig := db.Pool().Config()
	if poolConfig.MaxConns != 4 {
		t.Fatalf("expected max conns 4, got %d", poolConfig.MaxConns)
	}

	if poolConfig.MinConns != 0 {
		t.Fatalf("expected min conns 0, got %d", poolConfig.MinConns)
	}

	if poolConfig.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf("expected QueryExecModeExec, got %v", poolConfig.ConnConfig.DefaultQueryExecMode)
	}
}

func TestNewRejectsInvalidDatabaseURLSafely(t *testing.T) {
	_, err := New(context.Background(), Config{
		DatabaseURL: "postgresql://user@%zz",
		MaxConns:    4,
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	if strings.Contains(err.Error(), "postgresql://") {
		t.Fatal("expected error not to expose database password")
	}
}

func TestPingWithTestDatabaseURL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := New(context.Background(), Config{
		DatabaseURL: databaseURL,
		MaxConns:    1,
	})
	if err != nil {
		t.Fatalf("expected database to be configured, got %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("expected database ping to succeed, got %v", err)
	}
}
