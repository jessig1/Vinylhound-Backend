package concerts

import (
	"context"

	"vinylhound/shared/go/models"
)

// Store defines persistence operations for concerts
type Store interface {
	CreateConcert(ctx context.Context, token string, concert *models.Concert) (*models.Concert, error)
	ListConcertsByUser(ctx context.Context, token string, includeVenue bool) ([]*models.ConcertWithDetails, error)
	GetConcert(ctx context.Context, id int64) (*models.ConcertWithDetails, error)
	UpdateConcert(ctx context.Context, token string, id int64, concert *models.Concert) (*models.Concert, error)
	DeleteConcert(ctx context.Context, token string, id int64) error
	ListUpcomingConcerts(ctx context.Context, token string) ([]*models.ConcertWithDetails, error)
	ListConcertsByVenue(ctx context.Context, venueID int64) ([]*models.ConcertWithDetails, error)
	ListConcertsByArtist(ctx context.Context, token string, artistName string) ([]*models.ConcertWithDetails, error)
	MarkConcertAttended(ctx context.Context, token string, concertID int64, rating *int) error
}

// VenueService allows validating that venues exist before creating concerts
type VenueService interface {
	GetVenue(ctx context.Context, id int64) (*models.Venue, error)
}

// Service coordinates concert-related operations
type Service interface {
	Create(ctx context.Context, token string, concert *models.Concert) (*models.Concert, error)
	List(ctx context.Context, token string) ([]*models.ConcertWithDetails, error)
	Get(ctx context.Context, id int64) (*models.ConcertWithDetails, error)
	Update(ctx context.Context, token string, id int64, concert *models.Concert) (*models.Concert, error)
	Delete(ctx context.Context, token string, id int64) error
	ListUpcoming(ctx context.Context, token string) ([]*models.ConcertWithDetails, error)
	ListByVenue(ctx context.Context, venueID int64) ([]*models.ConcertWithDetails, error)
	ListByArtist(ctx context.Context, token string, artistName string) ([]*models.ConcertWithDetails, error)
	MarkAttended(ctx context.Context, token string, concertID int64, rating *int) error
}

type service struct {
	store        Store
	venueService VenueService // Optional: validate venues exist
}

// New constructs a concerts Service
func New(store Store, venueService VenueService) Service {
	return &service{
		store:        store,
		venueService: venueService,
	}
}

func (s *service) Create(ctx context.Context, token string, concert *models.Concert) (*models.Concert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Optional: Validate venue exists before creating concert
	if s.venueService != nil {
		_, err := s.venueService.GetVenue(ctx, concert.VenueID)
		if err != nil {
			return nil, err // Venue doesn't exist
		}
	}

	return s.store.CreateConcert(ctx, token, concert)
}

func (s *service) List(ctx context.Context, token string) ([]*models.ConcertWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListConcertsByUser(ctx, token, true)
}

func (s *service) Get(ctx context.Context, id int64) (*models.ConcertWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetConcert(ctx, id)
}

func (s *service) Update(ctx context.Context, token string, id int64, concert *models.Concert) (*models.Concert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Optional: Validate venue exists if being updated
	if s.venueService != nil && concert.VenueID > 0 {
		_, err := s.venueService.GetVenue(ctx, concert.VenueID)
		if err != nil {
			return nil, err
		}
	}

	return s.store.UpdateConcert(ctx, token, id, concert)
}

func (s *service) Delete(ctx context.Context, token string, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.DeleteConcert(ctx, token, id)
}

func (s *service) ListUpcoming(ctx context.Context, token string) ([]*models.ConcertWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListUpcomingConcerts(ctx, token)
}

func (s *service) ListByVenue(ctx context.Context, venueID int64) ([]*models.ConcertWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListConcertsByVenue(ctx, venueID)
}

func (s *service) ListByArtist(ctx context.Context, token string, artistName string) ([]*models.ConcertWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListConcertsByArtist(ctx, token, artistName)
}

func (s *service) MarkAttended(ctx context.Context, token string, concertID int64, rating *int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.MarkConcertAttended(ctx, token, concertID, rating)
}
