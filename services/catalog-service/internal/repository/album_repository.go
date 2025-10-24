package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// albumRepository handles album data persistence
type albumRepository struct {
	db *sql.DB
}

// NewAlbumRepository creates a new album repository
func NewAlbumRepository(db *sql.DB) AlbumRepository {
	return &albumRepository{db: db}
}

// CreateAlbum creates a new album
func (r *albumRepository) CreateAlbum(ctx context.Context, album *models.Album) (*models.Album, error) {
	now := time.Now()
	album.CreatedAt = now
	album.UpdatedAt = now

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO albums (title, artist, release_year, genre, cover_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, album.Title, album.Artist, album.ReleaseYear, album.Genre, album.CoverURL, album.CreatedAt, album.UpdatedAt).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("insert album: %w", err)
	}

	album.ID = id
	return album, nil
}

// GetAlbumByID retrieves an album by ID
func (r *albumRepository) GetAlbumByID(ctx context.Context, id int64) (*models.Album, error) {
	album := &models.Album{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, artist, release_year, genre, cover_url, created_at, updated_at
		FROM albums
		WHERE id = $1
	`, id).Scan(&album.ID, &album.Title, &album.Artist, &album.ReleaseYear, &album.Genre, &album.CoverURL, &album.CreatedAt, &album.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("album not found")
		}
		return nil, fmt.Errorf("get album: %w", err)
	}

	return album, nil
}

// ListAlbums retrieves albums with filtering
func (r *albumRepository) ListAlbums(ctx context.Context, filter models.AlbumFilter) ([]*models.Album, error) {
	query := `
		SELECT id, title, artist, release_year, genre, cover_url, created_at, updated_at
		FROM albums
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if filter.Artist != "" {
		query += fmt.Sprintf(" AND artist ILIKE $%d", argIndex)
		args = append(args, "%"+filter.Artist+"%")
		argIndex++
	}

	if filter.Genre != "" {
		query += fmt.Sprintf(" AND genre ILIKE $%d", argIndex)
		args = append(args, "%"+filter.Genre+"%")
		argIndex++
	}

	if filter.YearFrom > 0 {
		query += fmt.Sprintf(" AND release_year >= $%d", argIndex)
		args = append(args, filter.YearFrom)
		argIndex++
	}

	if filter.YearTo > 0 {
		query += fmt.Sprintf(" AND release_year <= $%d", argIndex)
		args = append(args, filter.YearTo)
		argIndex++
	}

	if filter.SearchTerm != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR artist ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.SearchTerm+"%")
		argIndex++
	}

	// Add ordering
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
		return nil, fmt.Errorf("query albums: %w", err)
	}
	defer rows.Close()

	var albums []*models.Album
	for rows.Next() {
		album := &models.Album{}
		err := rows.Scan(&album.ID, &album.Title, &album.Artist, &album.ReleaseYear, &album.Genre, &album.CoverURL, &album.CreatedAt, &album.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums: %w", err)
	}

	return albums, nil
}

// UpdateAlbum updates an existing album
func (r *albumRepository) UpdateAlbum(ctx context.Context, id int64, album *models.Album) (*models.Album, error) {
	album.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE albums
		SET title = $1, artist = $2, release_year = $3, genre = $4, cover_url = $5, updated_at = $6
		WHERE id = $7
	`, album.Title, album.Artist, album.ReleaseYear, album.Genre, album.CoverURL, album.UpdatedAt, id)

	if err != nil {
		return nil, fmt.Errorf("update album: %w", err)
	}

	album.ID = id
	return album, nil
}

// DeleteAlbum deletes an album
func (r *albumRepository) DeleteAlbum(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM albums
		WHERE id = $1
	`, id)

	if err != nil {
		return fmt.Errorf("delete album: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("album not found")
	}

	return nil
}
