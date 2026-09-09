package database

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotConfigured = errors.New("database not configured")
	ErrInvalidConfig = errors.New("database configuration error")
)

type Config struct {
	DatabaseURL string
	MaxConns    int32
}

type Database struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*Database, error) {
	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	if databaseURL == "" {
		return &Database{}, nil
	}

	if cfg.MaxConns <= 0 {
		return nil, ErrInvalidConfig
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, ErrInvalidConfig
	}

	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, ErrInvalidConfig
	}

	return &Database{pool: pool}, nil
}

func (db *Database) Configured() bool {
	return db != nil && db.pool != nil
}

func (db *Database) Ping(ctx context.Context) error {
	if !db.Configured() {
		return ErrNotConfigured
	}

	return db.pool.Ping(ctx)
}

func (db *Database) Close() {
	if db.Configured() {
		db.pool.Close()
	}
}

func (db *Database) Pool() *pgxpool.Pool {
	if db == nil {
		return nil
	}

	return db.pool
}
