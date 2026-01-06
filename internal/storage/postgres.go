package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saisaravanan/healing-eval/internal/config"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

func NewPostgresDB(ctx context.Context, cfg *config.DatabaseConfig) (*PostgresDB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &PostgresDB{Pool: pool}, nil
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
}

