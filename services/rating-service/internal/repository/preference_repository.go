package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// preferenceRepository handles user preference data persistence
type preferenceRepository struct {
	db *sql.DB
}

// NewPreferenceRepository creates a new preference repository
func NewPreferenceRepository(db *sql.DB) PreferenceRepository {
	return &preferenceRepository{db: db}
}

// GetUserPreferences retrieves user genre preferences
func (r *preferenceRepository) GetUserPreferences(ctx context.Context, userID int64) ([]*models.GenrePreference, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, genre, weight, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
		ORDER BY weight DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query preferences: %w", err)
	}
	defer rows.Close()

	var preferences []*models.GenrePreference
	for rows.Next() {
		pref := &models.GenrePreference{}
		err := rows.Scan(&pref.ID, &pref.UserID, &pref.Genre, &pref.Weight, &pref.CreatedAt, &pref.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan preference: %w", err)
		}
		preferences = append(preferences, pref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preferences: %w", err)
	}

	return preferences, nil
}

// UpdateUserPreferences updates user genre preferences
func (r *preferenceRepository) UpdateUserPreferences(ctx context.Context, userID int64, preferences []*models.GenrePreference) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing preferences
	_, err = tx.ExecContext(ctx, `
		DELETE FROM user_preferences
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("delete preferences: %w", err)
	}

	// Insert new preferences
	now := time.Now()
	for _, pref := range preferences {
		pref.UserID = userID
		pref.CreatedAt = now
		pref.UpdatedAt = now

		_, err := tx.ExecContext(ctx, `
			INSERT INTO user_preferences (user_id, genre, weight, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`, pref.UserID, pref.Genre, pref.Weight, pref.CreatedAt, pref.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert preference: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
