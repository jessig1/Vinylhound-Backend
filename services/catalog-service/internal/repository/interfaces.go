package repository

import (
	"context"

	"vinylhound/shared/models"
)

// AlbumRepository defines the interface for album data operations
type AlbumRepository interface {
	CreateAlbum(ctx context.Context, album *models.Album) (*models.Album, error)
	GetAlbumByID(ctx context.Context, id int64) (*models.Album, error)
	ListAlbums(ctx context.Context, filter models.AlbumFilter) ([]*models.Album, error)
	UpdateAlbum(ctx context.Context, id int64, album *models.Album) (*models.Album, error)
	DeleteAlbum(ctx context.Context, id int64) error
}

// ArtistRepository defines the interface for artist data operations
type ArtistRepository interface {
	GetArtistByID(ctx context.Context, id int64) (*models.Artist, error)
	ListArtists(ctx context.Context, name string) ([]*models.Artist, error)
}

// SongRepository defines the interface for song data operations
type SongRepository interface {
	GetSongByID(ctx context.Context, id int64) (*models.Song, error)
	ListSongs(ctx context.Context, albumID int64, artist string) ([]*models.Song, error)
}
