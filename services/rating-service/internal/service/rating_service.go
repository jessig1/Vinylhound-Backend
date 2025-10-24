package service

import (
	"context"
	"fmt"

	"vinylhound/rating-service/internal/repository"
	"vinylhound/shared/models"
)

// RatingService handles rating-related business logic
type RatingService struct {
	repo repository.RatingRepository
}

// NewRatingService creates a new rating service
func NewRatingService(repo repository.RatingRepository) *RatingService {
	return &RatingService{repo: repo}
}

// CreateRating creates a new rating
func (s *RatingService) CreateRating(ctx context.Context, rating *models.Rating) (*models.Rating, error) {
	// Validate rating value
	if rating.Rating < 1 || rating.Rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	createdRating, err := s.repo.CreateRating(ctx, rating)
	if err != nil {
		return nil, fmt.Errorf("create rating: %w", err)
	}
	return createdRating, nil
}

// GetRating retrieves a rating by ID
func (s *RatingService) GetRating(ctx context.Context, id int64) (*models.Rating, error) {
	rating, err := s.repo.GetRatingByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get rating: %w", err)
	}
	return rating, nil
}

// ListRatings retrieves ratings with optional filtering
func (s *RatingService) ListRatings(ctx context.Context, filter models.RatingFilter) ([]*models.Rating, error) {
	ratings, err := s.repo.ListRatings(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list ratings: %w", err)
	}
	return ratings, nil
}

// UpdateRating updates an existing rating
func (s *RatingService) UpdateRating(ctx context.Context, id int64, rating *models.Rating) (*models.Rating, error) {
	// Validate rating value
	if rating.Rating < 1 || rating.Rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	updatedRating, err := s.repo.UpdateRating(ctx, id, rating)
	if err != nil {
		return nil, fmt.Errorf("update rating: %w", err)
	}
	return updatedRating, nil
}

// DeleteRating deletes a rating
func (s *RatingService) DeleteRating(ctx context.Context, id int64) error {
	if err := s.repo.DeleteRating(ctx, id); err != nil {
		return fmt.Errorf("delete rating: %w", err)
	}
	return nil
}
