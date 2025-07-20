package listing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/storage"
	"github.com/ocenb/marketplace/internal/utils"
)

type ListingRepoInterface interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (storage.SqlTx, error)
	Create(ctx context.Context, userID int64, title string, description string, imageUrl string, price int64) (*models.Listing, error)
	GetFeed(ctx context.Context, userID int64, page, limit int, sortBy, sortOrder string, minPrice, maxPrice int64) (*models.ListingsFeed, error)
	CheckExists(ctx context.Context, id int64) (bool, error)
}

type ListingRepo struct {
	postgres *sql.DB
	log      *slog.Logger
}

func New(postgres *sql.DB, log *slog.Logger) ListingRepoInterface {
	return &ListingRepo{postgres, log}
}

func (r *ListingRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (storage.SqlTx, error) {
	return r.postgres.BeginTx(ctx, opts)
}

func (r *ListingRepo) Create(
	ctx context.Context,
	userID int64,
	title string,
	description string,
	imageUrl string,
	price int64,
) (*models.Listing, error) {
	query := `
		WITH inserted_listing AS (
			INSERT INTO listings (user_id, title, description, image_url, price)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, user_id, title, description, image_url, price, created_at
		)
		SELECT
			il.id,
			il.user_id,
			u.login AS author_login,
			il.title,
			il.description,
			il.image_url,
			il.price,
			il.created_at
		FROM
			inserted_listing AS il
		JOIN
			users AS u ON il.user_id = u.id;
	`

	listing := models.Listing{IsOwner: true}
	row := storage.QueryRowWithTx(ctx, r.postgres, query, userID, title, description, imageUrl, price)

	err := row.Scan(
		&listing.ID,
		&listing.UserID,
		&listing.AuthorLogin,
		&listing.Title,
		&listing.Description,
		&listing.ImageURL,
		&listing.Price,
		&listing.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("listing creation failed or author not found: %w", err)
		}
		return nil, fmt.Errorf("failed to scan created listing with author login: %w", err)
	}

	return &listing, nil
}

func (r *ListingRepo) GetFeed(ctx context.Context, userID int64, page, limit int, sortBy, sortOrder string, minPrice, maxPrice int64) (*models.ListingsFeed, error) {
	var whereClauses []string
	var args []any
	argCounter := 1

	if minPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("l.price >= $%d", argCounter))
		args = append(args, minPrice)
		argCounter++
	}
	if maxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("l.price <= $%d", argCounter))
		args = append(args, maxPrice)
		argCounter++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	orderByClause := "ORDER BY l.created_at DESC"
	switch sortBy {
	case "createdAt":
		orderByClause = fmt.Sprintf("ORDER BY l.created_at %s", strings.ToUpper(sortOrder))
	case "price":
		orderByClause = fmt.Sprintf("ORDER BY l.price %s", strings.ToUpper(sortOrder))
	}

	offset := (page - 1) * limit

	mainQuery := fmt.Sprintf(`
		SELECT
			l.id,
			l.user_id,
			u.login AS author_login,
			l.title,
			l.description,
			l.image_url,
			l.price,
			l.created_at
		FROM
			listings AS l
		JOIN
			users AS u ON l.user_id = u.id
		%s
		%s
		LIMIT $%d OFFSET $%d;
	`, whereClause, orderByClause, argCounter, argCounter+1)

	queryArgs := make([]any, len(args))
	copy(queryArgs, args)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := storage.QueryWithTx(ctx, r.postgres, mainQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query listing feed: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("Failed to close rows", utils.ErrLog(err))
		}
	}()

	var listingsFeed models.ListingsFeed
	for rows.Next() {
		var listing models.Listing
		err := rows.Scan(
			&listing.ID,
			&listing.UserID,
			&listing.AuthorLogin,
			&listing.Title,
			&listing.Description,
			&listing.ImageURL,
			&listing.Price,
			&listing.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan listing row: %w", err)
		}
		if userID > 0 {
			listing.IsOwner = listing.UserID == userID
		}
		listingsFeed.Listings = append(listingsFeed.Listings, listing)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	listingsFeed.Total = len(listingsFeed.Listings)
	listingsFeed.Page = page
	listingsFeed.Limit = limit

	return &listingsFeed, nil
}

func (r *ListingRepo) CheckExists(ctx context.Context, id int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM listing WHERE id = $1)`
	var exists bool
	err := storage.QueryRowWithTx(ctx, r.postgres, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
