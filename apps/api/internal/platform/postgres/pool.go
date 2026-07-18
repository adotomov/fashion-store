package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxConns        = int32(10)
	minConns        = int32(2)
	maxConnLifetime = time.Hour
	maxConnIdleTime = 30 * time.Minute
	connectTimeout  = 5 * time.Second
)

// NewPool creates a connection pool to PostgreSQL using the given DSN,
// applying sensible pool limits and verifying connectivity with a ping. When
// traceQueries is true every query is wrapped in an OTel span (exported to
// Cloud Trace) via otelpgx, so a request's DB calls appear in its trace
// waterfall.
func NewPool(ctx context.Context, databaseURL string, traceQueries bool) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.MaxConnIdleTime = maxConnIdleTime

	if traceQueries {
		cfg.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithTrimSQLInSpanName(),
		)
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
