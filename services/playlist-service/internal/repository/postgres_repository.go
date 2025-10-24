package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// PostgresRepository persists playlists in PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a repository backed by PostgreSQL.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// List returns all playlists ordered by creation time (newest first).
func (r *PostgresRepository) List(ctx context.Context) ([]*models.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, owner, created_at, updated_at
		FROM playlists
		ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var playlists []*models.Playlist
	for rows.Next() {
		var playlist models.Playlist
		if err := rows.Scan(&playlist.ID, &playlist.Title, &playlist.Owner, &playlist.CreatedAt, &playlist.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		songs, err := r.listSongs(ctx, playlist.ID)
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

// Get returns a single playlist.
func (r *PostgresRepository) Get(ctx context.Context, id int64) (*models.Playlist, error) {
	var playlist models.Playlist
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, owner, created_at, updated_at
		FROM playlists
		WHERE id = $1`, id).Scan(&playlist.ID, &playlist.Title, &playlist.Owner, &playlist.CreatedAt, &playlist.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlaylistNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	songs, err := r.listSongs(ctx, playlist.ID)
	if err != nil {
		return nil, err
	}
	playlist.Songs = songs
	playlist.SongCount = len(songs)
	return &playlist, nil
}

// Create persists a playlist and its songs.
func (r *PostgresRepository) Create(ctx context.Context, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO playlists (title, owner, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		RETURNING id, created_at, updated_at`,
		playlist.Title, playlist.Owner, now,
	).Scan(&playlist.ID, &playlist.CreatedAt, &playlist.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert playlist: %w", err)
	}

	if err = r.replaceSongsTx(ctx, tx, playlist.ID, playlist.Songs); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit playlist create: %w", err)
	}

	playlist.SongCount = len(playlist.Songs)
	return clonePlaylist(playlist), nil
}

// Update replaces a playlist and its songs.
func (r *PostgresRepository) Update(ctx context.Context, id int64, playlist *models.Playlist) (*models.Playlist, error) {
	if playlist == nil {
		return nil, errors.New("playlist is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `
		UPDATE playlists
		SET title = $1, owner = $2, updated_at = $3
		WHERE id = $4`,
		playlist.Title, playlist.Owner, time.Now().UTC(), id)
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

	if err = r.replaceSongsTx(ctx, tx, id, playlist.Songs); err != nil {
		return nil, err
	}

	var updated models.Playlist
	if err = tx.QueryRowContext(ctx, `
		SELECT id, title, owner, created_at, updated_at
		FROM playlists
		WHERE id = $1`, id).Scan(&updated.ID, &updated.Title, &updated.Owner, &updated.CreatedAt, &updated.UpdatedAt); err != nil {
		return nil, fmt.Errorf("reload playlist: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit playlist update: %w", err)
	}

	songs := make([]models.PlaylistSong, len(playlist.Songs))
	copy(songs, playlist.Songs)
	updated.Songs = songs
	updated.SongCount = len(songs)
	return &updated, nil
}

// Delete removes a playlist.
func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1`, id)
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

func (r *PostgresRepository) listSongs(ctx context.Context, playlistID int64) ([]models.PlaylistSong, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, artist, album, length_seconds, genre
		FROM playlist_songs
		WHERE playlist_id = $1
		ORDER BY position ASC, id ASC`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("list playlist songs: %w", err)
	}
	defer rows.Close()

	var songs []models.PlaylistSong
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

func (r *PostgresRepository) replaceSongsTx(ctx context.Context, tx *sql.Tx, playlistID int64, songs []models.PlaylistSong) (err error) {
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
