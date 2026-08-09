package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	database := &Pool{pool: pool}
	if err := database.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("verify PostgreSQL/PostGIS connection: %w", err)
	}
	return database, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	var version string
	if err := p.pool.QueryRow(ctx, "SELECT PostGIS_Version()").Scan(&version); err != nil {
		return fmt.Errorf("query PostGIS version: %w", err)
	}
	return nil
}

func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

func (p *Pool) Close() {
	p.pool.Close()
}
