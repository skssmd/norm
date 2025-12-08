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
	cfg,err:=pgxpool.ParseConfig(dsn)
	if err!=nil{
		return nil,err
	}
	cfg.MaxConns=10
		// Baseline defaults — will tune later
	cfg.MaxConns = 20
	cfg.MinConns = 5
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 2 * time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err	
	}

	return &PGPool{Pool: pool}, nil

}