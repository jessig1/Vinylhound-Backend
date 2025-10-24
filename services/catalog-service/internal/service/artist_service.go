package service

import (
	"context"
	"fmt"

	"vinylhound/catalog-service/internal/repository"
	"vinylhound/shared/models"
)

// ArtistService handles artist-related business logic
type ArtistService struct {
	repo repository.ArtistRepository
}

// NewArtistService creates a new artist service
func NewArtistService(repo repository.ArtistRepository) *ArtistService {
	return &ArtistService{repo: repo}
}

// GetArtist retrieves an artist by ID
func (s *ArtistService) GetArtist(ctx context.Context, id int64) (*models.Artist, error) {
	artist, err := s.repo.GetArtistByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get artist: %w", err)
	}
	return artist, nil
}

// ListArtists retrieves artists with optional filtering
func (s *ArtistService) ListArtists(ctx context.Context, name string) ([]*models.Artist, error) {
	artists, err := s.repo.ListArtists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("list artists: %w", err)
	}
	return artists, nil
}
