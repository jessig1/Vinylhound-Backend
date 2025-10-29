package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vinylhound/shared/go/models"
)

var (
	ErrFavoriteAlreadyExists = errors.New("favorite already exists")
	ErrFavoriteNotFound      = errors.New("favorite not found")
	ErrInvalidFavoriteType   = errors.New("must specify either song_id or album_id, but not both")
)

// AddFavorite adds a song or album to the user's favorites.
// When a favorite is added, it also adds the item to the user's favorites playlist.
func (s *Store) AddFavorite(ctx context.Context, token string, songID *int64, albumID *int64) (*models.Favorite, error) {
	// Validate input
	if (songID == nil && albumID == nil) || (songID != nil && albumID != nil) {
		return nil, ErrInvalidFavoriteType
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

	// Insert favorite
	var favorite models.Favorite
	err = tx.QueryRowContext(ctx, `
		INSERT INTO favorites (user_id, song_id, album_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, song_id, album_id, created_at`,
		userID, songID, albumID, time.Now().UTC(),
	).Scan(&favorite.ID, &favorite.UserID, &favorite.SongID, &favorite.AlbumID, &favorite.CreatedAt)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			return nil, ErrFavoriteAlreadyExists
		}
		return nil, fmt.Errorf("insert favorite: %w", err)
	}

	// Get the user's favorites playlist
	var favPlaylistID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM playlists WHERE user_id = $1 AND is_favorite = TRUE
	`, userID).Scan(&favPlaylistID)
	if err != nil {
		return nil, fmt.Errorf("get favorites playlist: %w", err)
	}

	// If it's a song, add it directly to the favorites playlist
	if songID != nil {
		// Get song details
		var title, artist string
		var albumID sql.NullInt64
		var duration sql.NullInt32
		err = tx.QueryRowContext(ctx, `
			SELECT title, artist, album_id, duration
			FROM songs WHERE id = $1`, *songID).Scan(&title, &artist, &albumID, &duration)
		if err != nil {
			return nil, fmt.Errorf("get song details: %w", err)
		}

		// Get album name if exists
		var album string
		if albumID.Valid {
			tx.QueryRowContext(ctx, `SELECT title FROM albums WHERE id = $1`, albumID.Int64).Scan(&album)
		}

		// Check if song already exists in playlist
		var existingID int64
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM playlist_songs
			WHERE playlist_id = $1 AND title = $2 AND artist = $3
		`, favPlaylistID, title, artist).Scan(&existingID)

		// Only add if not already in playlist
		if errors.Is(err, sql.ErrNoRows) {
			// Get next position
			var maxPos sql.NullInt32
			tx.QueryRowContext(ctx, `SELECT MAX(position) FROM playlist_songs WHERE playlist_id = $1`, favPlaylistID).Scan(&maxPos)
			position := 0
			if maxPos.Valid {
				position = int(maxPos.Int32) + 1
			}

			var durationInt int
			if duration.Valid {
				durationInt = int(duration.Int32)
			}

			// Insert song into favorites playlist
			_, err = tx.ExecContext(ctx, `
				INSERT INTO playlist_songs (playlist_id, position, title, artist, album, length_seconds, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
				favPlaylistID, position, title, artist, nullIfEmpty(album), durationInt, time.Now().UTC())
			if err != nil {
				return nil, fmt.Errorf("insert song into favorites playlist: %w", err)
			}
		}
	}

	// If it's an album, add all songs from the album to the favorites playlist
	if albumID != nil {
		// Get all songs from the album
		rows, err := tx.QueryContext(ctx, `
			SELECT id, title, artist, duration
			FROM songs WHERE album_id = $1
			ORDER BY track_num ASC`, *albumID)
		if err != nil {
			return nil, fmt.Errorf("get album songs: %w", err)
		}
		defer rows.Close()

		// Get album name
		var albumName string
		tx.QueryRowContext(ctx, `SELECT title FROM albums WHERE id = $1`, *albumID).Scan(&albumName)

		// Get current max position
		var maxPos sql.NullInt32
		tx.QueryRowContext(ctx, `SELECT MAX(position) FROM playlist_songs WHERE playlist_id = $1`, favPlaylistID).Scan(&maxPos)
		position := 0
		if maxPos.Valid {
			position = int(maxPos.Int32) + 1
		}

		// Add each song to the favorites playlist
		for rows.Next() {
			var songID int64
			var title, artist string
			var duration sql.NullInt32
			if err := rows.Scan(&songID, &title, &artist, &duration); err != nil {
				return nil, fmt.Errorf("scan album song: %w", err)
			}

			// Check if song already exists in playlist
			var existingID int64
			err = tx.QueryRowContext(ctx, `
				SELECT id FROM playlist_songs
				WHERE playlist_id = $1 AND title = $2 AND artist = $3
			`, favPlaylistID, title, artist).Scan(&existingID)

			// Only add if not already in playlist
			if errors.Is(err, sql.ErrNoRows) {
				var durationInt int
				if duration.Valid {
					durationInt = int(duration.Int32)
				}

				_, err = tx.ExecContext(ctx, `
					INSERT INTO playlist_songs (playlist_id, position, title, artist, album, length_seconds, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
					favPlaylistID, position, title, artist, nullIfEmpty(albumName), durationInt, time.Now().UTC())
				if err != nil {
					return nil, fmt.Errorf("insert album song into favorites playlist: %w", err)
				}
				position++
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit favorite: %w", err)
	}

	return &favorite, nil
}

// RemoveFavorite removes a song or album from the user's favorites.
// When a favorite is removed, it does NOT remove the item from the favorites playlist
// (as per requirements: "songs can still be removed" from favorites playlist manually).
func (s *Store) RemoveFavorite(ctx context.Context, token string, songID *int64, albumID *int64) error {
	// Validate input
	if (songID == nil && albumID == nil) || (songID != nil && albumID != nil) {
		return ErrInvalidFavoriteType
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	var res sql.Result
	if songID != nil {
		res, err = s.db.ExecContext(ctx, `
			DELETE FROM favorites
			WHERE user_id = $1 AND song_id = $2`, userID, *songID)
	} else {
		res, err = s.db.ExecContext(ctx, `
			DELETE FROM favorites
			WHERE user_id = $1 AND album_id = $2`, userID, *albumID)
	}

	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrFavoriteNotFound
	}

	return nil
}

// ListFavorites returns all favorites for a user.
func (s *Store) ListFavorites(ctx context.Context, token string) ([]*models.Favorite, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, song_id, album_id, created_at
		FROM favorites
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()

	var favorites []*models.Favorite
	for rows.Next() {
		var fav models.Favorite
		if err := rows.Scan(&fav.ID, &fav.UserID, &fav.SongID, &fav.AlbumID, &fav.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan favorite: %w", err)
		}
		favorites = append(favorites, &fav)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate favorites: %w", err)
	}

	return favorites, nil
}

// IsFavorite checks if a song or album is favorited by the user.
func (s *Store) IsFavorite(ctx context.Context, token string, songID *int64, albumID *int64) (bool, error) {
	// Validate input
	if (songID == nil && albumID == nil) || (songID != nil && albumID != nil) {
		return false, ErrInvalidFavoriteType
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return false, err
	}

	var exists bool
	if songID != nil {
		err = s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND song_id = $2)
		`, userID, *songID).Scan(&exists)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND album_id = $2)
		`, userID, *albumID).Scan(&exists)
	}

	if err != nil {
		return false, fmt.Errorf("check favorite: %w", err)
	}

	return exists, nil
}

// GetFavoritesPlaylist returns the user's favorites playlist.
func (s *Store) GetFavoritesPlaylist(ctx context.Context, token string) (*models.Playlist, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	var playlist models.Playlist
	var description sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT id, title, description, owner, user_id, created_at, updated_at, tags, is_public, is_favorite
		FROM playlists
		WHERE user_id = $1 AND is_favorite = TRUE`, userID).Scan(&playlist.ID, &playlist.Title, &description, &playlist.Owner, &playlist.UserID,
		&playlist.CreatedAt, &playlist.UpdatedAt, pq.Array(&playlist.Tags), &playlist.IsPublic, &playlist.IsFavorite)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("favorites playlist not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get favorites playlist: %w", err)
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
