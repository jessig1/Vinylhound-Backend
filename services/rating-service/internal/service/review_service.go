package service

import (
	"context"
	"fmt"

	"vinylhound/rating-service/internal/repository"
	"vinylhound/shared/models"
)

// ReviewService handles review-related business logic
type ReviewService struct {
	repo repository.ReviewRepository
}

// NewReviewService creates a new review service
func NewReviewService(repo repository.ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

// CreateReview creates a new review
func (s *ReviewService) CreateReview(ctx context.Context, review *models.Review) (*models.Review, error) {
	createdReview, err := s.repo.CreateReview(ctx, review)
	if err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}
	return createdReview, nil
}

// GetReview retrieves a review by ID
func (s *ReviewService) GetReview(ctx context.Context, id int64) (*models.Review, error) {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	return review, nil
}

// ListReviews retrieves reviews with optional filtering
func (s *ReviewService) ListReviews(ctx context.Context, userID, albumID int64) ([]*models.Review, error) {
	reviews, err := s.repo.ListReviews(ctx, userID, albumID)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	return reviews, nil
}

// UpdateReview updates an existing review
func (s *ReviewService) UpdateReview(ctx context.Context, id int64, review *models.Review) (*models.Review, error) {
	updatedReview, err := s.repo.UpdateReview(ctx, id, review)
	if err != nil {
		return nil, fmt.Errorf("update review: %w", err)
	}
	return updatedReview, nil
}

// DeleteReview deletes a review
func (s *ReviewService) DeleteReview(ctx context.Context, id int64) error {
	if err := s.repo.DeleteReview(ctx, id); err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	return nil
}
