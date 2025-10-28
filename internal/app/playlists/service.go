package playlists

import (
	"context"

	"vinylhound/shared/go/models"
)

// Store captures the persistence needs for playlist workflows.
type Store interface {
	ListPlaylists(ctx context.Context, token string) ([]*models.Playlist, error)
	GetPlaylist(ctx context.Context, id int64) (*models.Playlist, error)
	CreatePlaylist(ctx context.Context, token string, playlist *models.Playlist) (*models.Playlist, error)
	UpdatePlaylist(ctx context.Context, token string, id int64, playlist *models.Playlist) (*models.Playlist, error)
	DeletePlaylist(ctx context.Context, token string, id int64) error
	AddSongToPlaylist(ctx context.Context, token string, playlistID int64, songID int64) error
	RemoveSongFromPlaylist(ctx context.Context, token string, playlistID int64, songID int64) error
}

// Service coordinates playlist-related operations.
type Service interface {
	List(ctx context.Context, token string) ([]*models.Playlist, error)
	Get(ctx context.Context, id int64) (*models.Playlist, error)
	Create(ctx context.Context, token string, playlist *models.Playlist) (*models.Playlist, error)
	Update(ctx context.Context, token string, id int64, playlist *models.Playlist) (*models.Playlist, error)
	Delete(ctx context.Context, token string, id int64) error
	AddSong(ctx context.Context, token string, playlistID int64, songID int64) error
	RemoveSong(ctx context.Context, token string, playlistID int64, songID int64) error
}

type service struct {
	store Store
}

// New constructs a Service backed by the provided Store.
func New(store Store) Service {
	return &service{store: store}
}

func (s *service) List(ctx context.Context, token string) ([]*models.Playlist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListPlaylists(ctx, token)
}

func (s *service) Get(ctx context.Context, id int64) (*models.Playlist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetPlaylist(ctx, id)
}

func (s *service) Create(ctx context.Context, token string, playlist *models.Playlist) (*models.Playlist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.CreatePlaylist(ctx, token, playlist)
}

func (s *service) Update(ctx context.Context, token string, id int64, playlist *models.Playlist) (*models.Playlist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.UpdatePlaylist(ctx, token, id, playlist)
}

func (s *service) Delete(ctx context.Context, token string, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.DeletePlaylist(ctx, token, id)
}

func (s *service) AddSong(ctx context.Context, token string, playlistID int64, songID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.AddSongToPlaylist(ctx, token, playlistID, songID)
}

func (s *service) RemoveSong(ctx context.Context, token string, playlistID int64, songID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.RemoveSongFromPlaylist(ctx, token, playlistID, songID)
}
