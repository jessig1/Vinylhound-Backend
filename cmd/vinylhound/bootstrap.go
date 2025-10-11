package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"vinylhound/internal/store"
)

func bootstrapDemoData(ctx context.Context, db *sql.DB, dataStore *store.Store) error {
	if err := ensureDemoUser(dataStore); err != nil {
		return err
	}
	if err := ensureDemoAlbums(ctx, db); err != nil {
		return err
	}
	return nil
}

func ensureDemoUser(dataStore *store.Store) error {
	if err := dataStore.CreateUser("demo", "demo123", []string{
		"Welcome to Vinylhound!",
		"Start by customizing your personal playlist.",
	}); err != nil && !errors.Is(err, store.ErrUserExists) {
		return fmt.Errorf("bootstrap demo user: %w", err)
	}
	return nil
}

func ensureDemoAlbums(ctx context.Context, db *sql.DB) error {
	const username = "demo"

	albumsTableExists, err := tableExists(ctx, db, "albums")
	if err != nil {
		return fmt.Errorf("check albums table: %w", err)
	}
	if !albumsTableExists {
		return nil
	}

	var userID int64
	if err := db.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE username = $1
	`, username).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lookup demo user: %w", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM albums
		WHERE user_id = $1
	`, userID).Scan(&count); err != nil {
		return fmt.Errorf("count demo albums: %w", err)
	}
	if count > 0 {
		return nil
	}

	type seedAlbum struct {
		Artist     string
		Title      string
		Year       int
		Tracks     []string
		Genres     []string
		Rating     int
		UserRating *int
		Favorited  bool
	}

	intPtr := func(v int) *int { return &v }

	albums := []seedAlbum{
		{
			Artist:     "Boards of Canada",
			Title:      "Music Has the Right to Children",
			Year:       1998,
			Tracks:     []string{"Turquoise Hexagon Sun", "Roygbiv", "Aquarius"},
			Genres:     []string{"Electronic", "Ambient"},
			Rating:     5,
			UserRating: intPtr(5),
			Favorited:  true,
		},
		{
			Artist:     "Massive Attack",
			Title:      "Mezzanine",
			Year:       1998,
			Tracks:     []string{"Angel", "Teardrop", "Inertia Creeps"},
			Genres:     []string{"Trip Hop"},
			Rating:     4,
			UserRating: intPtr(4),
			Favorited:  true,
		},
		{
			Artist:     "Portishead",
			Title:      "Dummy",
			Year:       1994,
			Tracks:     []string{"Mysterons", "Sour Times", "Glory Box"},
			Genres:     []string{"Trip Hop"},
			Rating:     5,
			UserRating: intPtr(5),
			Favorited:  true,
		},
		{
			Artist:     "Radiohead",
			Title:      "OK Computer",
			Year:       1997,
			Tracks:     []string{"Airbag", "Paranoid Android", "No Surprises"},
			Genres:     []string{"Alternative Rock"},
			Rating:     5,
			UserRating: intPtr(5),
			Favorited:  true,
		},
		{
			Artist:     "Nightmares on Wax",
			Title:      "Carboot Soul",
			Year:       1999,
			Tracks:     []string{"Les Nuits", "Morse", "Finer"},
			Genres:     []string{"Downtempo", "Electronic"},
			Rating:     4,
			UserRating: intPtr(4),
			Favorited:  false,
		},
		{
			Artist:     "Bonobo",
			Title:      "Migration",
			Year:       2017,
			Tracks:     []string{"Migration", "Break Apart", "Kerala"},
			Genres:     []string{"Electronic", "Downtempo"},
			Rating:     4,
			UserRating: intPtr(4),
			Favorited:  false,
		},
		{
			Artist:     "Nils Frahm",
			Title:      "Spaces",
			Year:       2013,
			Tracks:     []string{"An Aborted Beginning", "Says", "Hammers"},
			Genres:     []string{"Modern Classical"},
			Rating:     5,
			UserRating: intPtr(5),
			Favorited:  true,
		},
		{
			Artist:     "Thundercat",
			Title:      "Drunk",
			Year:       2017,
			Tracks:     []string{"Uh Uh", "Them Changes", "Show You The Way"},
			Genres:     []string{"Funk", "Jazz"},
			Rating:     4,
			UserRating: intPtr(3),
			Favorited:  false,
		},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	preferenceTableExists, err := tableExists(ctx, tx, "user_album_preferences")
	if err != nil {
		return fmt.Errorf("check user_album_preferences table: %w", err)
	}

	for _, album := range albums {
		tracksJSON, err := json.Marshal(album.Tracks)
		if err != nil {
			return fmt.Errorf("marshal tracks for %q: %w", album.Title, err)
		}
		genresJSON, err := json.Marshal(album.Genres)
		if err != nil {
			return fmt.Errorf("marshal genres for %q: %w", album.Title, err)
		}

		var albumID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO albums (user_id, artist, title, release_year, tracks, genres, rating)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
			RETURNING id
		`, userID, album.Artist, album.Title, album.Year, string(tracksJSON), string(genresJSON), album.Rating).Scan(&albumID); err != nil {
			return fmt.Errorf("insert demo album %q: %w", album.Title, err)
		}

		if !preferenceTableExists || (album.UserRating == nil && !album.Favorited) {
			continue
		}

		var rating any
		if album.UserRating != nil {
			rating = *album.UserRating
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_album_preferences (user_id, album_id, rating, favorited, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_id, album_id)
			DO UPDATE SET rating = EXCLUDED.rating, favorited = EXCLUDED.favorited, updated_at = NOW()
		`, userID, albumID, rating, album.Favorited); err != nil {
			return fmt.Errorf("insert demo album preference for %q: %w", album.Title, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}
	tx = nil

	return nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func tableExists(ctx context.Context, q queryRower, table string) (bool, error) {
	var name sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT to_regclass($1)`, table).Scan(&name); err != nil {
		return false, err
	}
	return name.Valid, nil
}
