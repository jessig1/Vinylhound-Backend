package places

import (
	"context"

	"vinylhound/shared/go/models"
)

// Store defines persistence operations for places (venues & retailers)
type Store interface {
	// Venue operations
	CreateVenue(ctx context.Context, token string, venue *models.Venue) (*models.Venue, error)
	ListVenuesByUser(ctx context.Context, token string) ([]*models.Venue, error)
	GetVenue(ctx context.Context, id int64) (*models.Venue, error)
	UpdateVenue(ctx context.Context, token string, id int64, venue *models.Venue) (*models.Venue, error)
	DeleteVenue(ctx context.Context, token string, id int64) error

	// Retailer operations
	CreateRetailer(ctx context.Context, token string, retailer *models.Retailer) (*models.Retailer, error)
	ListRetailersByUser(ctx context.Context, token string) ([]*models.Retailer, error)
	GetRetailer(ctx context.Context, id int64) (*models.Retailer, error)
	UpdateRetailer(ctx context.Context, token string, id int64, retailer *models.Retailer) (*models.Retailer, error)
	DeleteRetailer(ctx context.Context, token string, id int64) error
}

// Service coordinates place-related operations (venues and retailers)
type Service interface {
	// Venue operations
	CreateVenue(ctx context.Context, token string, venue *models.Venue) (*models.Venue, error)
	ListVenues(ctx context.Context, token string) ([]*models.Venue, error)
	GetVenue(ctx context.Context, id int64) (*models.Venue, error)
	UpdateVenue(ctx context.Context, token string, id int64, venue *models.Venue) (*models.Venue, error)
	DeleteVenue(ctx context.Context, token string, id int64) error

	// Retailer operations
	CreateRetailer(ctx context.Context, token string, retailer *models.Retailer) (*models.Retailer, error)
	ListRetailers(ctx context.Context, token string) ([]*models.Retailer, error)
	GetRetailer(ctx context.Context, id int64) (*models.Retailer, error)
	UpdateRetailer(ctx context.Context, token string, id int64, retailer *models.Retailer) (*models.Retailer, error)
	DeleteRetailer(ctx context.Context, token string, id int64) error
}

type service struct {
	store Store
}

// New constructs a places Service backed by the provided Store
func New(store Store) Service {
	return &service{store: store}
}

// Venue implementations
func (s *service) CreateVenue(ctx context.Context, token string, venue *models.Venue) (*models.Venue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.CreateVenue(ctx, token, venue)
}

func (s *service) ListVenues(ctx context.Context, token string) ([]*models.Venue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListVenuesByUser(ctx, token)
}

func (s *service) GetVenue(ctx context.Context, id int64) (*models.Venue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetVenue(ctx, id)
}

func (s *service) UpdateVenue(ctx context.Context, token string, id int64, venue *models.Venue) (*models.Venue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.UpdateVenue(ctx, token, id, venue)
}

func (s *service) DeleteVenue(ctx context.Context, token string, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.DeleteVenue(ctx, token, id)
}

// Retailer implementations
func (s *service) CreateRetailer(ctx context.Context, token string, retailer *models.Retailer) (*models.Retailer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.CreateRetailer(ctx, token, retailer)
}

func (s *service) ListRetailers(ctx context.Context, token string) ([]*models.Retailer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListRetailersByUser(ctx, token)
}

func (s *service) GetRetailer(ctx context.Context, id int64) (*models.Retailer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetRetailer(ctx, id)
}

func (s *service) UpdateRetailer(ctx context.Context, token string, id int64, retailer *models.Retailer) (*models.Retailer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.UpdateRetailer(ctx, token, id, retailer)
}

func (s *service) DeleteRetailer(ctx context.Context, token string, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.DeleteRetailer(ctx, token, id)
}
