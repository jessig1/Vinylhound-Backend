package service

import (
	"context"
	"fmt"

	"vinylhound/catalog-service/internal/repository"
	"vinylhound/shared/models"
)

// AlbumService handles album-related business logic
type AlbumService struct {
	repo repository.AlbumRepository
}

// NewAlbumService creates a new album service
func NewAlbumService(repo repository.AlbumRepository) *AlbumService {
	return &AlbumService{repo: repo}
}

// CreateAlbum creates a new album
func (s *AlbumService) CreateAlbum(ctx context.Context, album *models.Album) (*models.Album, error) {
	createdAlbum, err := s.repo.CreateAlbum(ctx, album)
	if err != nil {
		return nil, fmt.Errorf("create album: %w", err)
	}
	return createdAlbum, nil
}

// GetAlbum retrieves an album by ID
func (s *AlbumService) GetAlbum(ctx context.Context, id int64) (*models.Album, error) {
	album, err := s.repo.GetAlbumByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get album: %w", err)
	}
	return album, nil
}

// ListAlbums retrieves albums with optional filtering
func (s *AlbumService) ListAlbums(ctx context.Context, filter models.AlbumFilter) ([]*models.Album, error) {
	albums, err := s.repo.ListAlbums(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list albums: %w", err)
	}
	return albums, nil
}

// UpdateAlbum updates an existing album
func (s *AlbumService) UpdateAlbum(ctx context.Context, id int64, album *models.Album) (*models.Album, error) {
	updatedAlbum, err := s.repo.UpdateAlbum(ctx, id, album)
	if err != nil {
		return nil, fmt.Errorf("update album: %w", err)
	}
	return updatedAlbum, nil
}

// DeleteAlbum deletes an album
func (s *AlbumService) DeleteAlbum(ctx context.Context, id int64) error {
	if err := s.repo.DeleteAlbum(ctx, id); err != nil {
		return fmt.Errorf("delete album: %w", err)
	}
	return nil
}
