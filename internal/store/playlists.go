package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"vinylhound/shared/go/models"
)

var ErrPlaylistNotFound = errors.New("playlist not found")

// ListPlaylists returns all playlists for a user (by token).
func (s *Store) ListPlaylists(ctx context.Context, token string) ([]*models.Playlist, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, description, owner, user_id, created_at, updated_at, tags, is_public
		FROM playlists
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var playlists []*models.Playlist
	for rows.Next() {
		var playlist models.Playlist
		var description sql.NullString
		if err := rows.Scan(&playlist.ID, &playlist.Title, &description, &playlist.Owner, &playlist.UserID,
			&playlist.CreatedAt, &playlist.UpdatedAt, pq.Array(&playlist.Tags), &playlist.IsPublic); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		playlist.Description = description.String

		songs, err := s.listPlaylistSongs(ctx, playlist.ID)
		if err != nil {
			return nil, err
		}
		playlist.Songs = songs
		playlist.SongCount = len(songs)
		playlists = append(playlists, &playlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlists: %w", err)
	}
	return playlists, nil
}

// GetPlaylist returns a single playlist by ID.
func (s *Store) GetPlaylist(ctx context.Context, id int64) (*models.Playlist, error) {
	var playlist models.Playlist
	var description sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, description, owner, user_id, created_at, updated_at, tags, is_public
		FROM playlists
		WHERE id = $1`, id).Scan(&playlist.ID, &playlist.Title, &description, &playlist.Owner, &playlist.UserID,
		&playlist.CreatedAt, &playlist.UpdatedAt, pq.Array(&playlist.Tags), &playlist.IsPublic)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlaylistNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	playlist.Description = description.String

	songs, err := s.listPlaylistSongs(ctx, playlist.ID)
	if err != nil {
		return nil, err
	}
	playlist.Songs = songs
	playlist.SongCount = len(songs)
	return &playlist, nil
}

// CreatePlaylist persists a new playlist.
func (s *Store) CreatePlaylist(ctx context.Context, token string, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Get username for the owner field
	var username string
	if err := s.db.QueryRowContext(ctx, `SELECT username FROM users WHERE id = $1`, userID).Scan(&username); err != nil {
		return nil, fmt.Errorf("get username: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()

	var description sql.NullString
	if playlist.Description != "" {
		description = sql.NullString{String: playlist.Description, Valid: true}
	}

	if err = tx.QueryRowContext(ctx, `
		INSERT INTO playlists (title, description, owner, user_id, created_at, updated_at, tags, is_public)
		VALUES ($1, $2, $3, $4, $5, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		playlist.Title, description, username, userID, now, pq.Array(playlist.Tags), playlist.IsPublic,
	).Scan(&playlist.ID, &playlist.CreatedAt, &playlist.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert playlist: %w", err)
	}

	playlist.Owner = username
	playlist.UserID = userID

	if err = s.replacePlaylistSongsTx(ctx, tx, playlist.ID, playlist.Songs); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit playlist create: %w", err)
	}

	playlist.SongCount = len(playlist.Songs)
	return playlist, nil
}

// UpdatePlaylist updates an existing playlist.
func (s *Store) UpdatePlaylist(ctx context.Context, token string, id int64, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var description sql.NullString
	if playlist.Description != "" {
		description = sql.NullString{String: playlist.Description, Valid: true}
	}

	// Don't update owner field - it should never change after creation
	res, err := tx.ExecContext(ctx, `
		UPDATE playlists
		SET title = $1, description = $2, updated_at = $3, tags = $4, is_public = $5
		WHERE id = $6 AND user_id = $7`,
		playlist.Title, description, time.Now().UTC(), pq.Array(playlist.Tags), playlist.IsPublic, id, userID)
	if err != nil {
		return nil, fmt.Errorf("update playlist: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return nil, ErrPlaylistNotFound
	}

	// Only replace songs if the input playlist has songs data
	// This prevents accidental deletion when updating just metadata
	if playlist.Songs != nil {
		if err = s.replacePlaylistSongsTx(ctx, tx, id, playlist.Songs); err != nil {
			return nil, err
		}
	}

	var updated models.Playlist
	var desc sql.NullString
	if err = tx.QueryRowContext(ctx, `
		SELECT id, title, description, owner, user_id, created_at, updated_at, tags, is_public
		FROM playlists
		WHERE id = $1`, id).Scan(&updated.ID, &updated.Title, &desc, &updated.Owner, &updated.UserID,
		&updated.CreatedAt, &updated.UpdatedAt, pq.Array(&updated.Tags), &updated.IsPublic); err != nil {
		return nil, fmt.Errorf("reload playlist: %w", err)
	}
	updated.Description = desc.String

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit playlist update: %w", err)
	}

	// Always fetch songs from database to ensure we have the current state
	songs, err := s.listPlaylistSongs(ctx, id)
	if err != nil {
		return nil, err
	}
	updated.Songs = songs
	updated.SongCount = len(songs)
	return &updated, nil
}

// DeletePlaylist removes a playlist.
func (s *Store) DeletePlaylist(ctx context.Context, token string, id int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrPlaylistNotFound
	}
	return nil
}

// AddSongToPlaylist adds a song to a playlist.
func (s *Store) AddSongToPlaylist(ctx context.Context, token string, playlistID int64, songID int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	// Verify playlist ownership
	var ownerID int64
	err = s.db.QueryRowContext(ctx, `SELECT user_id FROM playlists WHERE id = $1`, playlistID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlaylistNotFound
	}
	if err != nil {
		return fmt.Errorf("check playlist ownership: %w", err)
	}
	if ownerID != userID {
		return errors.New("not authorized to modify this playlist")
	}

	// Get song details
	var title, artist string
	var albumID sql.NullInt64
	var duration, trackNum sql.NullInt32
	err = s.db.QueryRowContext(ctx, `
		SELECT title, artist, album_id, duration, track_num
		FROM songs WHERE id = $1`, songID).Scan(&title, &artist, &albumID, &duration, &trackNum)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("song not found")
	}
	if err != nil {
		return fmt.Errorf("get song details: %w", err)
	}

	// Get album name if exists
	var album string
	if albumID.Valid {
		s.db.QueryRowContext(ctx, `SELECT title FROM albums WHERE id = $1`, albumID.Int64).Scan(&album)
	}

	// Get next position
	var maxPos sql.NullInt32
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM playlist_songs WHERE playlist_id = $1`, playlistID).Scan(&maxPos)
	position := 0
	if maxPos.Valid {
		position = int(maxPos.Int32) + 1
	}

	var durationInt int
	if duration.Valid {
		durationInt = int(duration.Int32)
	}

	// Insert song
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO playlist_songs (playlist_id, position, title, artist, album, length_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		playlistID, position, title, artist, nullIfEmpty(album), durationInt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert playlist song: %w", err)
	}

	return nil
}

// RemoveSongFromPlaylist removes a song from a playlist.
func (s *Store) RemoveSongFromPlaylist(ctx context.Context, token string, playlistID int64, songID int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	// Verify playlist ownership
	var ownerID int64
	err = s.db.QueryRowContext(ctx, `SELECT user_id FROM playlists WHERE id = $1`, playlistID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlaylistNotFound
	}
	if err != nil {
		return fmt.Errorf("check playlist ownership: %w", err)
	}
	if ownerID != userID {
		return errors.New("not authorized to modify this playlist")
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM playlist_songs
		WHERE playlist_id = $1 AND id = $2`, playlistID, songID)
	if err != nil {
		return fmt.Errorf("delete playlist song: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return errors.New("song not found in playlist")
	}

	return nil
}

func (s *Store) listPlaylistSongs(ctx context.Context, playlistID int64) ([]models.PlaylistSong, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, artist, COALESCE(album, ''), length_seconds, COALESCE(genre, '')
		FROM playlist_songs
		WHERE playlist_id = $1
		ORDER BY position ASC, id ASC`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("list playlist songs: %w", err)
	}
	defer rows.Close()

	songs := make([]models.PlaylistSong, 0)
	for rows.Next() {
		var song models.PlaylistSong
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.LengthSeconds, &song.Genre); err != nil {
			return nil, fmt.Errorf("scan playlist song: %w", err)
		}
		songs = append(songs, song)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlist songs: %w", err)
	}
	return songs, nil
}

func (s *Store) replacePlaylistSongsTx(ctx context.Context, tx *sql.Tx, playlistID int64, songs []models.PlaylistSong) (err error) {
	if _, err = tx.ExecContext(ctx, `DELETE FROM playlist_songs WHERE playlist_id = $1`, playlistID); err != nil {
		return fmt.Errorf("clear playlist songs: %w", err)
	}
	if len(songs) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO playlist_songs (playlist_id, position, title, artist, album, length_seconds, genre, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`)
	if err != nil {
		return fmt.Errorf("prepare insert playlist song: %w", err)
	}
	defer stmt.Close()

	for idx, song := range songs {
		if _, err = stmt.ExecContext(
			ctx,
			playlistID,
			idx,
			song.Title,
			song.Artist,
			nullIfEmpty(song.Album),
			song.LengthSeconds,
			nullIfEmpty(song.Genre),
			time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("insert playlist song: %w", err)
		}
	}
	return nil
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
