package search

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Store defines the persistence operations required by the search handler.
type Store interface {
	Search(ctx context.Context, query string, limit int) (Results, error)
}

// Results captures the different result buckets surfaced by the handler.
type Results struct {
	Artists []ArtistResult
	Albums  []AlbumResult
	Songs   []SongResult
}

// ArtistResult summarises an artist match.
type ArtistResult struct {
	ID         string
	Name       string
	AlbumCount int
	Href       string
	ImageURL   string
}

// AlbumResult summarises an album match.
type AlbumResult struct {
	ID          int64
	Title       string
	Artist      string
	ReleaseYear int
	Href        string
	ImageURL    string
}

// SongResult summarises a song match.
type SongResult struct {
	ID       int64
	Title    string
	Artist   string
	Album    string
	Href     string
	ImageURL string
}

// PGStore implements Store using PostgreSQL.
type PGStore struct {
	db *sql.DB
}

// NewPGStore creates a Store backed by the supplied database handle.
func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

// Search performs a fan-out query across artists, albums, and songs.
func (s *PGStore) Search(ctx context.Context, query string, limit int) (Results, error) {
	if limit <= 0 {
		limit = 10
	}
	like := "%" + query + "%"

	artists, err := s.fetchArtists(ctx, like, limit)
	if err != nil {
		return Results{}, err
	}

	albums, err := s.fetchAlbums(ctx, like, limit)
	if err != nil {
		return Results{}, err
	}

	songs, err := s.fetchSongs(ctx, like, limit)
	if err != nil {
		return Results{}, err
	}

	return Results{
		Artists: artists,
		Albums:  albums,
		Songs:   songs,
	}, nil
}

func (s *PGStore) fetchArtists(ctx context.Context, like string, limit int) ([]ArtistResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT artist, COUNT(*) AS album_count
		FROM albums
		WHERE artist ILIKE $1
		GROUP BY artist
		ORDER BY album_count DESC, artist ASC
		LIMIT $2
	`, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search artists: %w", err)
	}
	defer rows.Close()

	results := make([]ArtistResult, 0)
	for rows.Next() {
		var (
			name  string
			count int
		)
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}

		results = append(results, ArtistResult{
			ID:         makeArtistID(name),
			Name:       name,
			AlbumCount: count,
			Href:       "/api/v1/albums?artist=" + url.QueryEscape(name),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists: %w", err)
	}

	return results, nil
}

func (s *PGStore) fetchAlbums(ctx context.Context, like string, limit int) ([]AlbumResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, artist, COALESCE(release_year, 0)
		FROM albums
		WHERE title ILIKE $1 OR artist ILIKE $1
		ORDER BY release_year DESC, title ASC
		LIMIT $2
	`, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search albums: %w", err)
	}
	defer rows.Close()

	results := make([]AlbumResult, 0)
	for rows.Next() {
		var (
			id          int64
			title       string
			artist      string
			releaseYear int
		)

		if err := rows.Scan(&id, &title, &artist, &releaseYear); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}

		results = append(results, AlbumResult{
			ID:          id,
			Title:       title,
			Artist:      artist,
			ReleaseYear: releaseYear,
			Href:        "/api/v1/albums/" + strconv.FormatInt(id, 10),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums: %w", err)
	}

	return results, nil
}

func (s *PGStore) fetchSongs(ctx context.Context, like string, limit int) ([]SongResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.title, s.artist, COALESCE(a.title, '')
		FROM songs s
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE s.title ILIKE $1 OR s.artist ILIKE $1
		ORDER BY s.title ASC
		LIMIT $2
	`, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search songs: %w", err)
	}
	defer rows.Close()

	results := make([]SongResult, 0)
	for rows.Next() {
		var (
			id     int64
			title  string
			artist string
			album  string
		)
		if err := rows.Scan(&id, &title, &artist, &album); err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}

		results = append(results, SongResult{
			ID:     id,
			Title:  title,
			Artist: artist,
			Album:  album,
			Href:   "/api/v1/songs/" + strconv.FormatInt(id, 10),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate songs: %w", err)
	}

	return results, nil
}

func makeArtistID(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '_':
			return '-'
		default:
			return -1
		}
	}, slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "artist"
	}
	return "artist:" + slug
}
