package service

import (
	"context"
	"fmt"

	"vinylhound/catalog-service/internal/repository"
	"vinylhound/shared/models"
)

// SongService handles song-related business logic
type SongService struct {
	repo repository.SongRepository
}

// NewSongService creates a new song service
func NewSongService(repo repository.SongRepository) *SongService {
	return &SongService{repo: repo}
}

// GetSong retrieves a song by ID
func (s *SongService) GetSong(ctx context.Context, id int64) (*models.Song, error) {
	song, err := s.repo.GetSongByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get song: %w", err)
	}
	return song, nil
}

// ListSongs retrieves songs with optional filtering
func (s *SongService) ListSongs(ctx context.Context, albumID int64, artist string) ([]*models.Song, error) {
	songs, err := s.repo.ListSongs(ctx, albumID, artist)
	if err != nil {
		return nil, fmt.Errorf("list songs: %w", err)
	}
	return songs, nil
}
