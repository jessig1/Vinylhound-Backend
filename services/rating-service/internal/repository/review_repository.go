package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// reviewRepository handles review data persistence
type reviewRepository struct {
	db *sql.DB
}

// NewReviewRepository creates a new review repository
func NewReviewRepository(db *sql.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

// CreateReview creates a new review
func (r *reviewRepository) CreateReview(ctx context.Context, review *models.Review) (*models.Review, error) {
	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO reviews (user_id, album_id, title, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, review.UserID, review.AlbumID, review.Title, review.Content, review.CreatedAt, review.UpdatedAt).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("insert review: %w", err)
	}

	review.ID = id
	return review, nil
}

// GetReviewByID retrieves a review by ID
func (r *reviewRepository) GetReviewByID(ctx context.Context, id int64) (*models.Review, error) {
	review := &models.Review{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, album_id, title, content, created_at, updated_at
		FROM reviews
		WHERE id = $1
	`, id).Scan(&review.ID, &review.UserID, &review.AlbumID, &review.Title, &review.Content, &review.CreatedAt, &review.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("review not found")
		}
		return nil, fmt.Errorf("get review: %w", err)
	}

	return review, nil
}

// ListReviews retrieves reviews with filtering
func (r *reviewRepository) ListReviews(ctx context.Context, userID, albumID int64) ([]*models.Review, error) {
	query := `
		SELECT id, user_id, album_id, title, content, created_at, updated_at
		FROM reviews
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if userID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if albumID > 0 {
		query += fmt.Sprintf(" AND album_id = $%d", argIndex)
		args = append(args, albumID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*models.Review
	for rows.Next() {
		review := &models.Review{}
		err := rows.Scan(&review.ID, &review.UserID, &review.AlbumID, &review.Title, &review.Content, &review.CreatedAt, &review.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reviews: %w", err)
	}

	return reviews, nil
}

// UpdateReview updates an existing review
func (r *reviewRepository) UpdateReview(ctx context.Context, id int64, review *models.Review) (*models.Review, error) {
	review.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE reviews
		SET title = $1, content = $2, updated_at = $3
		WHERE id = $4
	`, review.Title, review.Content, review.UpdatedAt, id)

	if err != nil {
		return nil, fmt.Errorf("update review: %w", err)
	}

	review.ID = id
	return review, nil
}

// DeleteReview deletes a review
func (r *reviewRepository) DeleteReview(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM reviews
		WHERE id = $1
	`, id)

	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("review not found")
	}

	return nil
}
