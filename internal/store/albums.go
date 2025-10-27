package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidAlbum indicates validation failure for album data.
	ErrInvalidAlbum = errors.New("invalid album")
	// ErrAlbumNotFound signals a missing album record.
	ErrAlbumNotFound = errors.New("album not found")
)

// Album models a music record owned by a specific user.
type Album struct {
	ID            int64    `json:"id"`
	Artist        string   `json:"artist"`
	Title         string   `json:"title"`
	ReleaseYear   int      `json:"releaseYear"`
	Tracks        []string `json:"trackList"`
	Genres        []string `json:"genreList"`
	Rating        int      `json:"rating"`
	AverageRating float64  `json:"averageRating,omitempty"`
	RatingCount   int      `json:"ratingCount,omitempty"`
}

// AlbumPreference captures a user's personal rating and favorite flag for an album.
type AlbumPreference struct {
	Album     Album `json:"album"`
	Rating    *int  `json:"rating,omitempty"`
	Favorited bool  `json:"favorited"`
}

// CreateAlbum inserts a new album for the user represented by the session token.
func (s *Store) CreateAlbum(token string, album Album) (Album, error) {
	if err := validateAlbum(album); err != nil {
		return Album{}, err
	}

	album.Artist = strings.TrimSpace(album.Artist)
	album.Title = strings.TrimSpace(album.Title)

	ctx := context.Background()

	userID, err := s.userIDForToken(ctx, token)
	if err != nil {
		return Album{}, err
	}

	tracksJSON, err := json.Marshal(album.Tracks)
	if err != nil {
		return Album{}, fmt.Errorf("prepare tracks payload: %w", err)
	}
	genresJSON, err := json.Marshal(album.Genres)
	if err != nil {
		return Album{}, fmt.Errorf("prepare genres payload: %w", err)
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO albums (user_id, artist, title, release_year, tracks, genres, rating)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
		RETURNING id
	`, userID, album.Artist, album.Title, album.ReleaseYear, string(tracksJSON), string(genresJSON), album.Rating).Scan(&id)
	if err != nil {
		return Album{}, fmt.Errorf("insert album: %w", err)
	}

	album.ID = id
	return album, nil
}

// AlbumsByToken lists albums for the authenticated user.
func (s *Store) AlbumsByToken(token string) ([]Album, error) {
	ctx := context.Background()

	userID, err := s.userIDForToken(ctx, token)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, artist, title, release_year, tracks, genres, rating
		FROM albums
		WHERE user_id = $1
		ORDER BY release_year DESC, id ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("select albums: %w", err)
	}
	defer rows.Close()

	albums, err := scanAlbumRows(rows)
	if err != nil {
		return nil, err
	}

	albums, err = s.applyAlbumRatingStats(ctx, albums)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums: %w", err)
	}

	return albums, nil
}

// AlbumFilter constrains the results returned by ListAlbums.
type AlbumFilter struct {
	Artist      string
	Title       string
	Genre       string
	ReleaseYear int
	Rating      int
}

// ListAlbums returns albums matching the provided filter.
func (s *Store) ListAlbums(filter AlbumFilter) ([]Album, error) {
	ctx := context.Background()

	query := `
		SELECT id, artist, title, release_year, tracks, genres, rating
		FROM albums
	`

	var (
		clauses []string
		args    []any
	)

	if artist := strings.TrimSpace(filter.Artist); artist != "" {
		args = append(args, "%"+artist+"%")
		clauses = append(clauses, fmt.Sprintf("artist ILIKE $%d", len(args)))
	}
	if title := strings.TrimSpace(filter.Title); title != "" {
		args = append(args, "%"+title+"%")
		clauses = append(clauses, fmt.Sprintf("title ILIKE $%d", len(args)))
	}
	if filter.ReleaseYear > 0 {
		args = append(args, filter.ReleaseYear)
		clauses = append(clauses, fmt.Sprintf("release_year = $%d", len(args)))
	}
	if filter.Rating > 0 {
		args = append(args, filter.Rating)
		clauses = append(clauses, fmt.Sprintf("rating = $%d", len(args)))
	}
	if genre := strings.TrimSpace(filter.Genre); genre != "" {
		genreJSON, err := json.Marshal([]string{genre})
		if err != nil {
			return nil, fmt.Errorf("marshal genre filter: %w", err)
		}
		args = append(args, string(genreJSON))
		clauses = append(clauses, fmt.Sprintf("genres @> $%d::jsonb", len(args)))
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	query += " ORDER BY release_year DESC, id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select albums: %w", err)
	}
	defer rows.Close()

	albums, err := scanAlbumRows(rows)
	if err != nil {
		return nil, err
	}

	albums, err = s.applyAlbumRatingStats(ctx, albums)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums: %w", err)
	}

	return albums, nil
}

// AlbumByID returns a single album by its identifier.
func (s *Store) AlbumByID(id int64) (Album, error) {
	ctx := context.Background()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, artist, title, release_year, tracks, genres, rating
		FROM albums
		WHERE id = $1
	`, id)

	album, err := scanAlbumRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Album{}, ErrAlbumNotFound
		}
		return Album{}, err
	}

	albums, err := s.applyAlbumRatingStats(ctx, []Album{album})
	if err != nil {
		return Album{}, err
	}
	return albums[0], nil
}

// UpsertAlbumPreference sets or updates the calling user's rating/favorite for an album.
func (s *Store) UpsertAlbumPreference(token string, albumID int64, rating *int, favorited bool) error {
	if rating != nil {
		if err := validateAlbumRating(*rating); err != nil {
			return err
		}
	}

	ctx := context.Background()

	userID, err := s.userIDForToken(ctx, token)
	if err != nil {
		return err
	}

	var exists int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM albums
		WHERE id = $1
	`, albumID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAlbumNotFound
		}
		return fmt.Errorf("lookup album: %w", err)
	}

	if rating == nil && !favorited {
		if _, err := s.db.ExecContext(ctx, `
			DELETE FROM user_album_preferences
			WHERE user_id = $1 AND album_id = $2
		`, userID, albumID); err != nil {
			return fmt.Errorf("delete album preference: %w", err)
		}
		return nil
	}

	var ratingArg any
	if rating != nil {
		ratingArg = *rating
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_album_preferences (user_id, album_id, rating, favorited, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, album_id)
		DO UPDATE SET rating = EXCLUDED.rating, favorited = EXCLUDED.favorited, updated_at = NOW()
	`, userID, albumID, ratingArg, favorited); err != nil {
		return fmt.Errorf("upsert album preference: %w", err)
	}

	return nil
}

// AlbumPreferencesByToken returns the user's albums with their ratings/favorites.
func (s *Store) AlbumPreferencesByToken(token string) ([]AlbumPreference, error) {
	ctx := context.Background()

	userID, err := s.userIDForToken(ctx, token)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			a.id, a.artist, a.title, a.release_year, a.tracks, a.genres, a.rating,
			p.rating, p.favorited
		FROM user_album_preferences p
		JOIN albums a ON a.id = p.album_id
		WHERE p.user_id = $1
		ORDER BY p.updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("select album preferences: %w", err)
	}
	defer rows.Close()

	var preferences []AlbumPreference
	for rows.Next() {
		var (
			a          Album
			tracksJSON []byte
			genresJSON []byte
			rating     sql.NullInt64
			fav        bool
		)

		if err := rows.Scan(
			&a.ID,
			&a.Artist,
			&a.Title,
			&a.ReleaseYear,
			&tracksJSON,
			&genresJSON,
			&a.Rating,
			&rating,
			&fav,
		); err != nil {
			return nil, fmt.Errorf("scan album preference: %w", err)
		}

		if err := json.Unmarshal(tracksJSON, &a.Tracks); err != nil {
			return nil, fmt.Errorf("decode tracks: %w", err)
		}
		if err := json.Unmarshal(genresJSON, &a.Genres); err != nil {
			return nil, fmt.Errorf("decode genres: %w", err)
		}

		pref := AlbumPreference{
			Album:     a,
			Favorited: fav,
		}
		if rating.Valid {
			val := int(rating.Int64)
			pref.Rating = &val
		}
		preferences = append(preferences, pref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate album preferences: %w", err)
	}

	albums := make([]Album, len(preferences))
	for i := range preferences {
		albums[i] = preferences[i].Album
	}

	albums, err = s.applyAlbumRatingStats(ctx, albums)
	if err != nil {
		return nil, err
	}

	for i := range preferences {
		preferences[i].Album.AverageRating = albums[i].AverageRating
		preferences[i].Album.RatingCount = albums[i].RatingCount
	}

	return preferences, nil
}

func validateAlbum(album Album) error {
	artist := strings.TrimSpace(album.Artist)
	title := strings.TrimSpace(album.Title)

	switch {
	case artist == "":
		return fmt.Errorf("%w: artist is required", ErrInvalidAlbum)
	case title == "":
		return fmt.Errorf("%w: title is required", ErrInvalidAlbum)
	case album.ReleaseYear <= 0:
		return fmt.Errorf("%w: release year must be positive", ErrInvalidAlbum)
	case album.Rating < 1 || album.Rating > 5:
		return fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidAlbum)
	}

	return nil
}

func validateAlbumRating(rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidAlbum)
	}
	return nil
}

func buildPlaceholders(count int, start int) []string {
	if count <= 0 {
		return nil
	}
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", start+i)
	}
	return placeholders
}

func (s *Store) fetchAlbumRatingStats(ctx context.Context, albumIDs []int64) (map[int64]albumRatingStat, error) {
	if len(albumIDs) == 0 {
		return map[int64]albumRatingStat{}, nil
	}

	uniqueIDs := make([]int64, 0, len(albumIDs))
	seen := make(map[int64]struct{}, len(albumIDs))
	for _, id := range albumIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	placeholders := buildPlaceholders(len(uniqueIDs), 1)
	args := make([]any, len(uniqueIDs))
	for i, id := range uniqueIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT album_id, AVG(rating)::double precision AS average_rating, COUNT(DISTINCT user_id) AS rating_count
		FROM user_album_preferences
		WHERE rating IS NOT NULL AND album_id IN (%s)
		GROUP BY album_id
	`, strings.Join(placeholders, ", "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate album ratings: %w", err)
	}
	defer rows.Close()

	stats := make(map[int64]albumRatingStat, len(uniqueIDs))
	for rows.Next() {
		var (
			albumID int64
			average sql.NullFloat64
			count   int64
		)
		if err := rows.Scan(&albumID, &average, &count); err != nil {
			return nil, fmt.Errorf("scan album rating stats: %w", err)
		}
		stat := albumRatingStat{
			count: int(count),
		}
		if average.Valid {
			stat.average = average.Float64
		}
		stats[albumID] = stat
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate album rating stats: %w", err)
	}
	return stats, nil
}

func (s *Store) applyAlbumRatingStats(ctx context.Context, albums []Album) ([]Album, error) {
	if len(albums) == 0 {
		return albums, nil
	}

	ids := make([]int64, len(albums))
	for i, album := range albums {
		ids[i] = album.ID
		albums[i].AverageRating = 0
		albums[i].RatingCount = 0
	}

	stats, err := s.fetchAlbumRatingStats(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range albums {
		if stat, ok := stats[albums[i].ID]; ok {
			albums[i].AverageRating = stat.average
			albums[i].RatingCount = stat.count
		}
	}
	return albums, nil
}

type albumRatingStat struct {
	average float64
	count   int
}

type albumScanner interface {
	Scan(dest ...any) error
}

func scanAlbumRow(scanner albumScanner) (Album, error) {
	var (
		a          Album
		tracksJSON []byte
		genresJSON []byte
	)

	if err := scanner.Scan(&a.ID, &a.Artist, &a.Title, &a.ReleaseYear, &tracksJSON, &genresJSON, &a.Rating); err != nil {
		return Album{}, fmt.Errorf("scan album: %w", err)
	}

	if err := json.Unmarshal(tracksJSON, &a.Tracks); err != nil {
		return Album{}, fmt.Errorf("decode tracks: %w", err)
	}
	if err := json.Unmarshal(genresJSON, &a.Genres); err != nil {
		return Album{}, fmt.Errorf("decode genres: %w", err)
	}

	return a, nil
}

func scanAlbumRows(rows *sql.Rows) ([]Album, error) {
	var albums []Album

	for rows.Next() {
		a, err := scanAlbumRow(rows)
		if err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}

	return albums, nil
}
