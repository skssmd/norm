package driver

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGPool struct {
	Pool *pgxpool.Pool
}

func Connect(dsn string) (*PGPool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Connection pool configuration
	cfg.MaxConns = 20
	cfg.MinConns = 5
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 2 * time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	// Test the connection immediately
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &PGPool{Pool: pool}, nil
}

// Close closes the connection pool
func (p *PGPool) Close() {
	p.Pool.Close()
}
