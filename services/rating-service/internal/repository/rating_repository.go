package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// ratingRepository handles rating data persistence
type ratingRepository struct {
	db *sql.DB
}

// NewRatingRepository creates a new rating repository
func NewRatingRepository(db *sql.DB) RatingRepository {
	return &ratingRepository{db: db}
}

// CreateRating creates a new rating
func (r *ratingRepository) CreateRating(ctx context.Context, rating *models.Rating) (*models.Rating, error) {
	now := time.Now()
	rating.CreatedAt = now
	rating.UpdatedAt = now

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO ratings (user_id, album_id, rating, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, rating.UserID, rating.AlbumID, rating.Rating, rating.CreatedAt, rating.UpdatedAt).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("insert rating: %w", err)
	}

	rating.ID = id
	return rating, nil
}

// GetRatingByID retrieves a rating by ID
func (r *ratingRepository) GetRatingByID(ctx context.Context, id int64) (*models.Rating, error) {
	rating := &models.Rating{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, album_id, rating, created_at, updated_at
		FROM ratings
		WHERE id = $1
	`, id).Scan(&rating.ID, &rating.UserID, &rating.AlbumID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rating not found")
		}
		return nil, fmt.Errorf("get rating: %w", err)
	}

	return rating, nil
}

// ListRatings retrieves ratings with filtering
func (r *ratingRepository) ListRatings(ctx context.Context, filter models.RatingFilter) ([]*models.Rating, error) {
	query := `
		SELECT id, user_id, album_id, rating, created_at, updated_at
		FROM ratings
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.UserID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, filter.UserID)
		argIndex++
	}

	if filter.AlbumID > 0 {
		query += fmt.Sprintf(" AND album_id = $%d", argIndex)
		args = append(args, filter.AlbumID)
		argIndex++
	}

	if filter.MinRating > 0 {
		query += fmt.Sprintf(" AND rating >= $%d", argIndex)
		args = append(args, filter.MinRating)
		argIndex++
	}

	if filter.MaxRating > 0 {
		query += fmt.Sprintf(" AND rating <= $%d", argIndex)
		args = append(args, filter.MaxRating)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query ratings: %w", err)
	}
	defer rows.Close()

	var ratings []*models.Rating
	for rows.Next() {
		rating := &models.Rating{}
		err := rows.Scan(&rating.ID, &rating.UserID, &rating.AlbumID, &rating.Rating, &rating.CreatedAt, &rating.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan rating: %w", err)
		}
		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ratings: %w", err)
	}

	return ratings, nil
}

// UpdateRating updates an existing rating
func (r *ratingRepository) UpdateRating(ctx context.Context, id int64, rating *models.Rating) (*models.Rating, error) {
	rating.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE ratings
		SET rating = $1, updated_at = $2
		WHERE id = $3
	`, rating.Rating, rating.UpdatedAt, id)

	if err != nil {
		return nil, fmt.Errorf("update rating: %w", err)
	}

	rating.ID = id
	return rating, nil
}

// DeleteRating deletes a rating
func (r *ratingRepository) DeleteRating(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM ratings
		WHERE id = $1
	`, id)

	if err != nil {
		return fmt.Errorf("delete rating: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rating not found")
	}

	return nil
}
