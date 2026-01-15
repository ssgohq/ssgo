package store

import (
	"context"

	"{{.Module}}/internal/data/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the database queries with transaction support
type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewStore creates a new Store
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Queries returns the underlying Queries
func (s *Store) Queries() *db.Queries {
	return s.queries
}

// ExecTx executes a function within a database transaction
func (s *Store) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := db.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit(ctx)
}

// Close closes the connection pool
func (s *Store) Close() {
	s.pool.Close()
}