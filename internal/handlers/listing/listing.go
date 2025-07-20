package listing

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/ocenb/marketplace/internal/services/listing"
	"github.com/ocenb/marketplace/internal/utils"
	"github.com/ocenb/marketplace/internal/utils/httputil"
)

type ListingHandlerInterface interface {
	Create(w http.ResponseWriter, r *http.Request)
	GetFeed(w http.ResponseWriter, r *http.Request)
	RegisterRoutes(optionalAuthRouter, authRouter chi.Router)
}

type CreateListingRequest struct {
	Title       string `json:"title" validate:"required,min=5,max=200"`
	Description string `json:"description" validate:"max=1000"`
	ImageURL    string `json:"image_url" validate:"required,url"`
	Price       int64  `json:"price" validate:"required,min=0,max=100000000000"`
}

type GetFeedParams struct {
	Page      int    `validate:"omitempty,min=1"`
	Limit     int    `validate:"omitempty,min=1,max=100"`
	SortBy    string `validate:"omitempty,oneof=createdAt price"`
	SortOrder string `validate:"omitempty,oneof=asc desc"`
	MinPrice  int64  `validate:"omitempty,min=0"`
	MaxPrice  int64  `validate:"omitempty,min=0,gtefield=MinPrice"`
}

type ListingHandler struct {
	listingService listing.ListingServiceInterface
	log            *slog.Logger
	validator      *validator.Validate
}

func New(listingService listing.ListingServiceInterface, log *slog.Logger, validator *validator.Validate) ListingHandlerInterface {
	return &ListingHandler{
		listingService,
		log,
		validator,
	}
}

// @Summary Create a new listing
// @Param listing body CreateListingRequest true "Listing creation data"
// @Security BearerAuth
// @Success 201 {object} models.Listing "Listing created successfully"
// @Failure 400 {object} httputil.ErrorResponse "Bad request"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /listing [post]
func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	log := h.log.With(utils.OpLog("ListingHandler.Create"))

	userID, ok := utils.GetInfoFromContext(r.Context(), log)
	if !ok {
		httputil.InternalError(w, log)
		return
	}

	var req CreateListingRequest
	if !httputil.DecodeAndValidate(w, r, &req, h.validator, log) {
		return
	}
	err := httputil.ValidateImage(log, req.ImageURL)
	if err != nil {
		log.Error("Failed to validate image", utils.ErrLog(err))
		httputil.BadRequestError(w, log, fmt.Sprintf("Validation failed: %s", err.Error()))
		return
	}

	log.Debug("Create listing request validated successfully",
		slog.String("title", req.Title),
	)

	newListing, err := h.listingService.Create(r.Context(), userID, req.Title, req.Description, req.ImageURL, req.Price)
	if err != nil {
		log.Error("Internal error during Create listing", utils.ErrLog(err))
		httputil.InternalError(w, log)
		return
	}

	log.Info("Listing created successfully",
		slog.Int64("listing_id", newListing.ID),
		slog.String("title", newListing.Title),
		slog.Time("created_at", newListing.CreatedAt),
	)

	httputil.WriteJSON(w, newListing, http.StatusCreated, log)
}

// @Summary Get a feed of listings
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Number of items per page" default(10) minimum(1) maximum(100)
// @Param sortBy query string false "Sort by field (createdAt or price)" Enums(createdAt, price) default(createdAt)
// @Param sortOrder query string false "Sort order (asc or desc)" Enums(asc, desc) default(desc)
// @Param minPrice query integer false "Minimum price in kopecks" minimum(0)
// @Param maxPrice query integer false "Maximum price in kopecks" minimum(0)
// @Security BearerAuth
// @Success 200 {object} models.ListingsFeed "Successfully retrieved listing feed"
// @Failure 400 {object} httputil.ErrorResponse "Bad request"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /listing/feed [get]
func (h *ListingHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	log := h.log.With(utils.OpLog("ListingHandler.GetFeed"))

	userID, _ := utils.GetInfoFromContext(r.Context(), log)

	params := GetFeedParams{
		Page:      1,
		Limit:     10,
		SortBy:    "createdAt",
		SortOrder: "desc",
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val >= 1 {
			params.Page = val
		} else {
			httputil.BadRequestError(w, log, "Invalid 'page' parameter")
			return
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val >= 1 && val <= 100 {
			params.Limit = val
		} else {
			httputil.BadRequestError(w, log, "Invalid 'limit' parameter (must be 1-100)")
			return
		}
	}

	if sb := r.URL.Query().Get("sortBy"); sb != "" {
		if sb == "createdAt" || sb == "price" {
			params.SortBy = sb
		} else {
			httputil.BadRequestError(w, log, "Invalid 'sortBy' parameter (must be 'createdAt' or 'price')")
			return
		}
	}

	if so := r.URL.Query().Get("sortOrder"); so != "" {
		if so == "asc" || so == "desc" {
			params.SortOrder = so
		} else {
			httputil.BadRequestError(w, log, "Invalid 'sortOrder' parameter (must be 'asc' or 'desc')")
			return
		}
	}

	if minP := r.URL.Query().Get("minPrice"); minP != "" {
		if val, err := strconv.ParseInt(minP, 10, 64); err == nil && val >= 0 {
			params.MinPrice = val
		} else {
			httputil.BadRequestError(w, log, "Invalid 'minPrice' parameter")
			return
		}
	}

	if maxP := r.URL.Query().Get("maxPrice"); maxP != "" {
		if val, err := strconv.ParseInt(maxP, 10, 64); err == nil && val >= 0 {
			params.MaxPrice = val
		} else {
			httputil.BadRequestError(w, log, "Invalid 'maxPrice' parameter")
			return
		}
	}

	if params.MaxPrice > 0 && params.MinPrice > 0 && params.MaxPrice < params.MinPrice {
		httputil.BadRequestError(w, log, "'maxPrice' cannot be less than 'minPrice'")
		return
	}

	feed, err := h.listingService.GetFeed(r.Context(), userID, params.Page,
		params.Limit,
		params.SortBy,
		params.SortOrder,
		params.MinPrice,
		params.MaxPrice)
	if err != nil {
		log.Error("Internal error during Get listing feed", utils.ErrLog(err))
		httputil.InternalError(w, log)
		return
	}

	log.Info("Successfully retrieved listing feed", slog.Int("total", feed.Total))

	httputil.WriteJSON(w, feed, http.StatusOK, log)
}

func (h *ListingHandler) RegisterRoutes(optionalAuthRouter, authRouter chi.Router) {
	authRouter.Post("/listing", h.Create)
	optionalAuthRouter.Get("/listing/feed", h.GetFeed)
}
