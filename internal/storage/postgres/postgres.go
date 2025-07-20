package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/ocenb/marketplace/internal/config"
)

func New(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Postgres.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime)

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}
