package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/ocenb/marketplace/internal/storage"
)

type AuthRepoInterface interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (storage.SqlTx, error)
	CheckTokenExists(ctx context.Context, token string) (bool, error)
	CreateToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
	DeleteExpiredTokens(ctx context.Context) error
}

type AuthRepo struct {
	postgres *sql.DB
}

func New(postgres *sql.DB) AuthRepoInterface {
	return &AuthRepo{postgres: postgres}
}

func (r *AuthRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (storage.SqlTx, error) {
	return r.postgres.BeginTx(ctx, opts)
}

func (r *AuthRepo) CheckTokenExists(ctx context.Context, token string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tokens WHERE token = $1)`
	var exists bool
	err := storage.QueryRowWithTx(ctx, r.postgres, query, token).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *AuthRepo) CreateToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	query := `INSERT INTO tokens (token, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := storage.ExecWithTx(ctx, r.postgres, query, token, userID, expiresAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRepo) DeleteExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM tokens WHERE expires_at < $1`
	_, err := storage.ExecWithTx(ctx, r.postgres, query, time.Now())
	if err != nil {
		return err
	}

	return err
}
