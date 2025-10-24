package repository

import (
	"context"

	"vinylhound/shared/models"
)

// RatingRepository defines the interface for rating data operations
type RatingRepository interface {
	CreateRating(ctx context.Context, rating *models.Rating) (*models.Rating, error)
	GetRatingByID(ctx context.Context, id int64) (*models.Rating, error)
	ListRatings(ctx context.Context, filter models.RatingFilter) ([]*models.Rating, error)
	UpdateRating(ctx context.Context, id int64, rating *models.Rating) (*models.Rating, error)
	DeleteRating(ctx context.Context, id int64) error
}

// ReviewRepository defines the interface for review data operations
type ReviewRepository interface {
	CreateReview(ctx context.Context, review *models.Review) (*models.Review, error)
	GetReviewByID(ctx context.Context, id int64) (*models.Review, error)
	ListReviews(ctx context.Context, userID, albumID int64) ([]*models.Review, error)
	UpdateReview(ctx context.Context, id int64, review *models.Review) (*models.Review, error)
	DeleteReview(ctx context.Context, id int64) error
}

// PreferenceRepository defines the interface for preference data operations
type PreferenceRepository interface {
	GetUserPreferences(ctx context.Context, userID int64) ([]*models.UserPreference, error)
	UpdateUserPreferences(ctx context.Context, userID int64, preferences []*models.UserPreference) error
}
