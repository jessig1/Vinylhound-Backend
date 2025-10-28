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

// SongStore exposes song database operations.
type SongStore interface {
	ListSongs(ctx context.Context, filter store.SongFilter) ([]store.Song, error)
	GetSong(ctx context.Context, id int64) (store.Song, error)
}

// Service exposes song-centric operations.
type Service interface {
	ListByAlbum(ctx context.Context, albumID int64) ([]Song, error)
	Search(ctx context.Context, filter store.SongFilter) ([]store.Song, error)
	Get(ctx context.Context, id int64) (store.Song, error)
}

type service struct {
	albums AlbumProvider
	store  SongStore
}

// New constructs a song Service backed by the provided providers.
func New(albums AlbumProvider, songStore SongStore) Service {
	return &service{
		albums: albums,
		store:  songStore,
	}
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

func (s *service) Search(ctx context.Context, filter store.SongFilter) ([]store.Song, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListSongs(ctx, filter)
}

func (s *service) Get(ctx context.Context, id int64) (store.Song, error) {
	if err := ctx.Err(); err != nil {
		return store.Song{}, err
	}
	return s.store.GetSong(ctx, id)
}
