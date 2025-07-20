package listing

import (
	"context"

	"github.com/ocenb/marketplace/internal/metrics"
	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/repos/listing"
	"github.com/ocenb/marketplace/internal/storage"
)

type ListingServiceInterface interface {
	Create(ctx context.Context, userID int64, title string, description string, imageUrl string, price int64) (*models.Listing, error)
	GetFeed(ctx context.Context, userID int64, page, limit int, sortBy, sortOrder string, minPrice, maxPrice int64) (*models.ListingsFeed, error)
	CheckExists(ctx context.Context, id int64) (bool, error)
}

type ListingService struct {
	listingRepo listing.ListingRepoInterface
	metrics     *metrics.Metrics
}

func New(listingRepo listing.ListingRepoInterface, metrics *metrics.Metrics) ListingServiceInterface {
	return &ListingService{
		listingRepo: listingRepo,
		metrics:     metrics,
	}
}

func (s *ListingService) Create(ctx context.Context, userID int64, title string, description string, imageUrl string, price int64) (*models.Listing, error) {
	var result *models.Listing

	err := storage.WithTransaction(ctx, s.listingRepo, func(txCtx context.Context) error {
		listing, err := s.listingRepo.Create(txCtx, userID, title, description, imageUrl, price)
		if err != nil {
			return err
		}

		result = listing
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.metrics.ListingsCounter.Inc()

	return result, nil
}

func (s *ListingService) GetFeed(ctx context.Context, userID int64, page, limit int, sortBy, sortOrder string, minPrice, maxPrice int64) (*models.ListingsFeed, error) {
	return s.listingRepo.GetFeed(ctx, userID, page, limit, sortBy, sortOrder, minPrice, maxPrice)
}

func (s *ListingService) CheckExists(ctx context.Context, id int64) (bool, error) {
	return s.listingRepo.CheckExists(ctx, id)
}
