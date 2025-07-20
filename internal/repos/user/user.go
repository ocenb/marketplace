package user

import (
	"context"
	"database/sql"

	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/storage"
)

type UserRepoInterface interface {
	Create(ctx context.Context, login, passwordHash string) (*models.UserPublic, error)
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	CheckExists(ctx context.Context, login string) (bool, error)
}

type UserRepo struct {
	postgres *sql.DB
}

func New(postgres *sql.DB) UserRepoInterface {
	return &UserRepo{postgres: postgres}
}

func (r *UserRepo) Create(ctx context.Context, login, passwordHash string) (*models.UserPublic, error) {
	query := `
		INSERT INTO users (login, password_hash) 
		VALUES ($1, $2)
		RETURNING id, login, created_at
	`

	var user models.UserPublic
	row := storage.QueryRowWithTx(ctx, r.postgres, query, login, passwordHash)
	if err := row.Scan(&user.ID, &user.Login, &user.CreatedAt); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`

	var user models.User
	err := storage.QueryRowWithTx(ctx, r.postgres, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := storage.QueryRowWithTx(ctx, r.postgres, query, id).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) CheckExists(ctx context.Context, login string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`
	var exists bool
	err := storage.QueryRowWithTx(ctx, r.postgres, query, login).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
