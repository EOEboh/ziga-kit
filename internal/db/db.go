package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a type alias so the rest of the app can import db.Pool
// without having to know about pgxpool directly.
type Pool = pgxpool.Pool

// Connect creates and validates a pgxpool connection pool.
// It blocks until the pool is healthy or the context is cancelled.
//
// Recommended usage: call once in main(), pass *db.Pool around via
// dependency injection into handlers/services — never use a global.
func Connect(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: failed to parse DATABASE_URL: %w", err)
	}

	// Pool tuning — sane defaults for a small SaaS backend.
	// Revisit when you start seeing contention under load.
	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: failed to create pool: %w", err)
	}

	// Ping to surface misconfiguration immediately at startup
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: could not reach database — is it running? %w", err)
	}

	return pool, nil
}
