package songs

import (
	"context"
	"strings"

	"vinylhound/internal/store"
)

// Song models a track within an album.
type Song struct {
	AlbumID     int64  `json:"albumId"`
	TrackNumber int    `json:"trackNumber"`
	Title       string `json:"title"`
}

// AlbumProvider exposes the album lookup required for track retrieval.
type AlbumProvider interface {
	Get(ctx context.Context, id int64) (store.Album, error)
}

// Service exposes song-centric operations.
type Service interface {
	ListByAlbum(ctx context.Context, albumID int64) ([]Song, error)
}

type service struct {
	albums AlbumProvider
}

// New constructs a song Service backed by the provided album provider.
func New(albums AlbumProvider) Service {
	return &service{albums: albums}
}

func (s *service) ListByAlbum(ctx context.Context, albumID int64) ([]Song, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	album, err := s.albums.Get(ctx, albumID)
	if err != nil {
		return nil, err
	}

	var tracks []Song
	for idx, title := range album.Tracks {
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		tracks = append(tracks, Song{
			AlbumID:     albumID,
			TrackNumber: idx + 1,
			Title:       title,
		})
	}
	return tracks, nil
}
