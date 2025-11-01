package favorites

import (
	"context"

	"vinylhound/internal/store"
	"vinylhound/shared/go/models"
)

// Store defines persistence operations required for favorites workflows.
type Store interface {
	AddFavorite(ctx context.Context, token string, songID *int64, albumID *int64) (*models.Favorite, error)
	RemoveFavorite(ctx context.Context, token string, songID *int64, albumID *int64) error
	ListFavoriteTracks(ctx context.Context, token string) ([]*models.Favorite, error)
}

// Service describes high level favorites operations used by HTTP handlers.
type Service interface {
	FavoriteTrack(ctx context.Context, token string, trackID int64) (*models.Favorite, bool, error)
	UnfavoriteTrack(ctx context.Context, token string, trackID int64) error
	ListTrackFavorites(ctx context.Context, token string) ([]*models.Favorite, error)
}

type service struct {
	store Store
}

// New constructs a favorites Service backed by the given store.
func New(st Store) Service {
	return &service{store: st}
}

func (s *service) FavoriteTrack(ctx context.Context, token string, trackID int64) (*models.Favorite, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	fav, err := s.store.AddFavorite(ctx, token, &trackID, nil)
	if err != nil {
		if err == store.ErrFavoriteAlreadyExists {
			return nil, false, nil
		}
		return nil, false, err
	}
	return fav, true, nil
}

func (s *service) UnfavoriteTrack(ctx context.Context, token string, trackID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.store.RemoveFavorite(ctx, token, &trackID, nil)
	if err == store.ErrFavoriteNotFound {
		return nil
	}
	return err
}

func (s *service) ListTrackFavorites(ctx context.Context, token string) ([]*models.Favorite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListFavoriteTracks(ctx, token)
}
