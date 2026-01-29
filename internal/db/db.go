// Package db implements a way to work with database
package db

import (
	"context"
	_ "database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"l0/internal/config"
)

// A DB is a wrapper for database pool
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new instance of DB using pool
func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool}
}

// NewDBWithConfig creates a new instance of DB based on the configuration file
func NewDBWithConfig(ctx context.Context, cfg *config.Config) (*DB, error) {
	if cfg == nil {
		return nil, errors.New("no config was provided")
	}
	dns := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, cfg.Database.SSLMode,
	)
	poolCfg, err := pgxpool.ParseConfig(dns)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConnections)
	poolCfg.MinConns = int32(cfg.Database.MinOpenConnections)
	poolCfg.MinIdleConns = int32(cfg.Database.MinIdleConnections)
	poolCfg.HealthCheckPeriod = cfg.Database.HealthCheckPeriod

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// Ping calls the pool's ping
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)

}

// WithTx wraps the function with database query in a transaction
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) (any, error)) (any, error) {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})

	if err != nil {
		return nil, err
	}

	defer func() { _ = tx.Rollback(ctx) }()
	res, err := fn(tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Close closes the connection to the pool
func (db *DB) Close() {
	db.pool.Close()
}