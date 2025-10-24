package repository

import (
	"context"
	"database/sql"
	"fmt"

	"vinylhound/shared/models"
)

// songRepository handles song data persistence
type songRepository struct {
	db *sql.DB
}

// NewSongRepository creates a new song repository
func NewSongRepository(db *sql.DB) SongRepository {
	return &songRepository{db: db}
}

// GetSongByID retrieves a song by ID
func (r *songRepository) GetSongByID(ctx context.Context, id int64) (*models.Song, error) {
	song := &models.Song{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, artist, album_id, duration, track_num, created_at, updated_at
		FROM songs
		WHERE id = $1
	`, id).Scan(&song.ID, &song.Title, &song.Artist, &song.AlbumID, &song.Duration, &song.TrackNum, &song.CreatedAt, &song.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("song not found")
		}
		return nil, fmt.Errorf("get song: %w", err)
	}

	return song, nil
}

// ListSongs retrieves songs with optional filtering
func (r *songRepository) ListSongs(ctx context.Context, albumID int64, artist string) ([]*models.Song, error) {
	query := `
		SELECT id, title, artist, album_id, duration, track_num, created_at, updated_at
		FROM songs
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if albumID > 0 {
		query += fmt.Sprintf(" AND album_id = $%d", argIndex)
		args = append(args, albumID)
		argIndex++
	}

	if artist != "" {
		query += fmt.Sprintf(" AND artist ILIKE $%d", argIndex)
		args = append(args, "%"+artist+"%")
		argIndex++
	}

	query += " ORDER BY album_id, track_num ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query songs: %w", err)
	}
	defer rows.Close()

	var songs []*models.Song
	for rows.Next() {
		song := &models.Song{}
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.AlbumID, &song.Duration, &song.TrackNum, &song.CreatedAt, &song.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate songs: %w", err)
	}

	return songs, nil
}
