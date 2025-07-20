package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type TxKey struct{}

type SqlTx interface {
	Commit() error
	Rollback() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type BeginTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (SqlTx, error)
}

func WithTransaction(ctx context.Context, repo BeginTx, fn func(txCtx context.Context) error) error {
	tx, err := repo.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	var txErr error
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && txErr == nil {
			txErr = fmt.Errorf("tx err: %v, rb err: %v", txErr, rbErr)
		}
	}()

	txCtx := context.WithValue(ctx, TxKey{}, tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func getTxFromContext(ctx context.Context) (SqlTx, bool) {
	tx, ok := ctx.Value(TxKey{}).(SqlTx)
	return tx, ok
}

func ExecWithTx(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	tx, ok := getTxFromContext(ctx)
	if ok {
		return tx.ExecContext(ctx, query, args...)
	}
	return db.ExecContext(ctx, query, args...)
}

func QueryRowWithTx(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	tx, ok := getTxFromContext(ctx)
	if ok {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return db.QueryRowContext(ctx, query, args...)
}

func QueryWithTx(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	tx, ok := getTxFromContext(ctx)
	if ok {
		return tx.QueryContext(ctx, query, args...)
	}
	return db.QueryContext(ctx, query, args...)
}
