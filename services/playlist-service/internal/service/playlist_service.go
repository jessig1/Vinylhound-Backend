package service

import (
	"context"
	"errors"
	"time"

	"vinylhound/playlist-service/internal/repository"
	"vinylhound/shared/models"
)

// PlaylistService coordinates playlist operations.
type PlaylistService struct {
	repo repository.PlaylistRepository
}

// New creates a PlaylistService.
func New(repo repository.PlaylistRepository) *PlaylistService {
	return &PlaylistService{repo: repo}
}

// List returns all playlists.
func (s *PlaylistService) List(ctx context.Context) ([]*models.Playlist, error) {
	return s.repo.List(ctx)
}

// Get returns a playlist by id.
func (s *PlaylistService) Get(ctx context.Context, id int64) (*models.Playlist, error) {
	return s.repo.Get(ctx, id)
}

// Create stores a new playlist.
func (s *PlaylistService) Create(ctx context.Context, playlist *models.Playlist) (*models.Playlist, error) {
	if err := validatePlaylist(playlist); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, playlist)
}

// Update replaces an existing playlist.
func (s *PlaylistService) Update(ctx context.Context, id int64, playlist *models.Playlist) (*models.Playlist, error) {
	if err := validatePlaylist(playlist); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, playlist)
}

// Delete removes a playlist.
func (s *PlaylistService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func validatePlaylist(playlist *models.Playlist) error {
	if playlist == nil {
		return errors.New("playlist is required")
	}
	if playlist.Title == "" {
		return errors.New("playlist title is required")
	}
	if playlist.Owner == "" {
		return errors.New("playlist owner is required")
	}
	if playlist.CreatedAt.IsZero() {
		playlist.CreatedAt = time.Now().UTC()
	}
	for i := range playlist.Songs {
		if playlist.Songs[i].Title == "" {
			return errors.New("song title is required")
		}
		if playlist.Songs[i].Artist == "" {
			return errors.New("song artist is required")
		}
		if playlist.Songs[i].LengthSeconds < 0 {
			return errors.New("song length_seconds must be non-negative")
		}
	}
	return nil
}
