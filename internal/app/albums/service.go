package albums

import (
	"context"

	"vinylhound/internal/store"
)

// Store captures the persistence needs for album workflows.
type Store interface {
	CreateAlbum(token string, album store.Album) (store.Album, error)
	AlbumsByToken(token string) ([]store.Album, error)
	ListAlbums(filter store.AlbumFilter) ([]store.Album, error)
	AlbumByID(id int64) (store.Album, error)
}

// Service coordinates album-related operations.
type Service interface {
	Create(ctx context.Context, token string, album store.Album) (store.Album, error)
	ListByUser(ctx context.Context, token string) ([]store.Album, error)
	List(ctx context.Context, filter store.AlbumFilter) ([]store.Album, error)
	Get(ctx context.Context, id int64) (store.Album, error)
}

type service struct {
	store Store
}

// New constructs a Service backed by the provided Store.
func New(store Store) Service {
	return &service{store: store}
}

func (s *service) Create(ctx context.Context, token string, album store.Album) (store.Album, error) {
	if err := ctx.Err(); err != nil {
		return store.Album{}, err
	}
	return s.store.CreateAlbum(token, album)
}

func (s *service) ListByUser(ctx context.Context, token string) ([]store.Album, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.AlbumsByToken(token)
}

func (s *service) List(ctx context.Context, filter store.AlbumFilter) ([]store.Album, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListAlbums(filter)
}

func (s *service) Get(ctx context.Context, id int64) (store.Album, error) {
	if err := ctx.Err(); err != nil {
		return store.Album{}, err
	}
	return s.store.AlbumByID(id)
}
