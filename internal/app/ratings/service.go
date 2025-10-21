package ratings

import (
	"context"

	"vinylhound/internal/store"
)

// Store defines the persistence hooks for ratings workflows.
type Store interface {
	UpsertAlbumPreference(token string, albumID int64, rating *int, favorited bool) error
	AlbumPreferencesByToken(token string) ([]store.AlbumPreference, error)
}

// Service coordinates rating updates and queries.
type Service interface {
	Upsert(ctx context.Context, token string, albumID int64, rating *int, favorited bool) error
	ListByUser(ctx context.Context, token string) ([]store.AlbumPreference, error)
}

type service struct {
	store Store
}

// New constructs a ratings Service backed by the given Store.
func New(store Store) Service {
	return &service{store: store}
}

func (s *service) Upsert(ctx context.Context, token string, albumID int64, rating *int, favorited bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.UpsertAlbumPreference(token, albumID, rating, favorited)
}

func (s *service) ListByUser(ctx context.Context, token string) ([]store.AlbumPreference, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.AlbumPreferencesByToken(token)
}
