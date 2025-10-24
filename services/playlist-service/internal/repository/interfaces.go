package repository

import (
	"context"

	"vinylhound/shared/models"
)

// PlaylistRepository defines storage operations for playlists.
type PlaylistRepository interface {
	List(ctx context.Context) ([]*models.Playlist, error)
	Get(ctx context.Context, id int64) (*models.Playlist, error)
	Create(ctx context.Context, playlist *models.Playlist) (*models.Playlist, error)
	Update(ctx context.Context, id int64, playlist *models.Playlist) (*models.Playlist, error)
	Delete(ctx context.Context, id int64) error
}

