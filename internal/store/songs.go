package store

import (
	"context"
	"database/sql"
	"fmt"
)

// Song represents a song/track in the database.
type Song struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	AlbumID  *int64 `json:"album_id,omitempty"`
	Album    string `json:"album,omitempty"`
	Duration int    `json:"duration,omitempty"`
	TrackNum int    `json:"track_num,omitempty"`
	Genre    string `json:"genre,omitempty"`
}

// SongFilter defines criteria for filtering songs.
type SongFilter struct {
	Query   string
	Artist  string
	Album   string
	AlbumID *int64
}

// Search is an alias for ListSongs for API compatibility
func (s *Store) Search(ctx context.Context, filter SongFilter) ([]Song, error) {
	return s.ListSongs(ctx, filter)
}

// ListSongs returns songs matching the filter.
func (s *Store) ListSongs(ctx context.Context, filter SongFilter) ([]Song, error) {
	query := `
		SELECT s.id, s.title, s.artist, s.album_id, COALESCE(a.title, '') as album,
		       COALESCE(s.duration, 0), COALESCE(s.track_num, 0)
		FROM songs s
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.Query != "" {
		query += fmt.Sprintf(" AND (s.title ILIKE $%d OR s.artist ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Query+"%")
		argIdx++
	}

	if filter.Artist != "" {
		query += fmt.Sprintf(" AND s.artist ILIKE $%d", argIdx)
		args = append(args, "%"+filter.Artist+"%")
		argIdx++
	}

	if filter.Album != "" {
		query += fmt.Sprintf(" AND a.title ILIKE $%d", argIdx)
		args = append(args, "%"+filter.Album+"%")
		argIdx++
	}

	if filter.AlbumID != nil {
		query += fmt.Sprintf(" AND s.album_id = $%d", argIdx)
		args = append(args, *filter.AlbumID)
		argIdx++
	}

	query += " ORDER BY s.album_id, s.track_num, s.title LIMIT 100"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query songs: %w", err)
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		var albumID sql.NullInt64
		var album string
		var duration, trackNum int

		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &albumID, &album, &duration, &trackNum); err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}

		if albumID.Valid {
			song.AlbumID = &albumID.Int64
			song.Album = album
		}
		song.Duration = duration
		song.TrackNum = trackNum

		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate songs: %w", err)
	}

	return songs, nil
}

// Get is an alias for GetSong for API compatibility
func (s *Store) Get(ctx context.Context, id int64) (Song, error) {
	return s.GetSong(ctx, id)
}

// GetSong returns a single song by ID.
func (s *Store) GetSong(ctx context.Context, id int64) (Song, error) {
	var song Song
	var albumID sql.NullInt64
	var album string
	var duration, trackNum sql.NullInt32

	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.title, s.artist, s.album_id, COALESCE(a.title, ''),
		       s.duration, s.track_num
		FROM songs s
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.id = $1`, id).Scan(&song.ID, &song.Title, &song.Artist, &albumID, &album, &duration, &trackNum)

	if err == sql.ErrNoRows {
		return Song{}, fmt.Errorf("song not found")
	}
	if err != nil {
		return Song{}, fmt.Errorf("get song: %w", err)
	}

	if albumID.Valid {
		song.AlbumID = &albumID.Int64
		song.Album = album
	}
	if duration.Valid {
		song.Duration = int(duration.Int32)
	}
	if trackNum.Valid {
		song.TrackNum = int(trackNum.Int32)
	}

	return song, nil
}
