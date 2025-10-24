package repository

import (
	"context"
	"database/sql"
	"fmt"

	"vinylhound/shared/models"
)

// artistRepository handles artist data persistence
type artistRepository struct {
	db *sql.DB
}

// NewArtistRepository creates a new artist repository
func NewArtistRepository(db *sql.DB) ArtistRepository {
	return &artistRepository{db: db}
}

// GetArtistByID retrieves an artist by ID
func (r *artistRepository) GetArtistByID(ctx context.Context, id int64) (*models.Artist, error) {
	artist := &models.Artist{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, biography, image_url, created_at, updated_at
		FROM artists
		WHERE id = $1
	`, id).Scan(&artist.ID, &artist.Name, &artist.Biography, &artist.ImageURL, &artist.CreatedAt, &artist.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist not found")
		}
		return nil, fmt.Errorf("get artist: %w", err)
	}

	return artist, nil
}

// ListArtists retrieves artists with optional name filtering
func (r *artistRepository) ListArtists(ctx context.Context, name string) ([]*models.Artist, error) {
	query := `
		SELECT id, name, biography, image_url, created_at, updated_at
		FROM artists
		WHERE 1=1
	`
	args := []interface{}{}

	if name != "" {
		query += " AND name ILIKE $1"
		args = append(args, "%"+name+"%")
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query artists: %w", err)
	}
	defer rows.Close()

	var artists []*models.Artist
	for rows.Next() {
		artist := &models.Artist{}
		err := rows.Scan(&artist.ID, &artist.Name, &artist.Biography, &artist.ImageURL, &artist.CreatedAt, &artist.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists: %w", err)
	}

	return artists, nil
}
