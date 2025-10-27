package service

import (
	"context"
	"fmt"

	"vinylhound/rating-service/internal/repository"
	"vinylhound/shared/models"
)

// PreferenceService handles user preference-related business logic
type PreferenceService struct {
	repo repository.PreferenceRepository
}

// NewPreferenceService creates a new preference service
func NewPreferenceService(repo repository.PreferenceRepository) *PreferenceService {
	return &PreferenceService{repo: repo}
}

// GetPreferences retrieves user genre preferences
func (s *PreferenceService) GetPreferences(ctx context.Context, userID int64) ([]*models.GenrePreference, error) {
	preferences, err := s.repo.GetUserPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	return preferences, nil
}

// UpdatePreferences updates user genre preferences
func (s *PreferenceService) UpdatePreferences(ctx context.Context, userID int64, preferences []*models.GenrePreference) error {
	if err := s.repo.UpdateUserPreferences(ctx, userID, preferences); err != nil {
		return fmt.Errorf("update preferences: %w", err)
	}
	return nil
}
