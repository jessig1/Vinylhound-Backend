package searchservice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"vinylhound/internal/musicapi"
	"vinylhound/internal/store"
)

// Service provides unified search across multiple music providers and stores results
type Service struct {
	db               *sql.DB
	spotifyClient    musicapi.MusicAPIClient
	appleMusicClient musicapi.MusicAPIClient
	store            *store.Store
}

// NewService creates a new search service
func NewService(db *sql.DB, spotifyClient, appleMusicClient musicapi.MusicAPIClient, st *store.Store) *Service {
	return &Service{
		db:               db,
		spotifyClient:    spotifyClient,
		appleMusicClient: appleMusicClient,
		store:            st,
	}
}

// SearchOptions defines search parameters
type SearchOptions struct {
	Query        string
	Type         string // "artist", "album", "track", or "all"
	Provider     string // "spotify", "apple_music", or "all"
	Limit        int
	StoreResults bool // Whether to store results in database
}

// SearchResults contains aggregated results from all providers
type SearchResults struct {
	Artists []musicapi.Artist `json:"artists"`
	Albums  []musicapi.Album  `json:"albums"`
	Tracks  []musicapi.Track  `json:"tracks"`
}

// Search performs a unified search across all configured providers
func (s *Service) Search(ctx context.Context, opts SearchOptions) (*SearchResults, error) {
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	results := &SearchResults{
		Artists: []musicapi.Artist{},
		Albums:  []musicapi.Album{},
		Tracks:  []musicapi.Track{},
	}

	// Use WaitGroup to search providers concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 2)

	// Search Spotify
	if opts.Provider == "" || opts.Provider == "all" || opts.Provider == "spotify" {
		if s.spotifyClient != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				spotifyResults, err := s.searchProvider(ctx, s.spotifyClient, opts)
				if err != nil {
					log.Printf("Spotify search error: %v", err)
					errChan <- fmt.Errorf("spotify: %w", err)
					return
				}

				mu.Lock()
				results.Artists = append(results.Artists, spotifyResults.Artists...)
				results.Albums = append(results.Albums, spotifyResults.Albums...)
				results.Tracks = append(results.Tracks, spotifyResults.Tracks...)
				mu.Unlock()
			}()
		}
	}

	// Search Apple Music
	if opts.Provider == "" || opts.Provider == "all" || opts.Provider == "apple_music" {
		if s.appleMusicClient != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				appleResults, err := s.searchProvider(ctx, s.appleMusicClient, opts)
				if err != nil {
					log.Printf("Apple Music search error: %v", err)
					errChan <- fmt.Errorf("apple music: %w", err)
					return
				}

				mu.Lock()
				results.Artists = append(results.Artists, appleResults.Artists...)
				results.Albums = append(results.Albums, appleResults.Albums...)
				results.Tracks = append(results.Tracks, appleResults.Tracks...)
				mu.Unlock()
			}()
		}
	}

	wg.Wait()
	close(errChan)

	// Collect any errors (but don't fail if one provider fails)
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// Store results if requested
	if opts.StoreResults {
		if err := s.storeResults(ctx, results); err != nil {
			log.Printf("Failed to store search results: %v", err)
		}
	}

	return results, nil
}

// searchProvider performs search on a specific provider
func (s *Service) searchProvider(ctx context.Context, client musicapi.MusicAPIClient, opts SearchOptions) (*SearchResults, error) {
	results := &SearchResults{
		Artists: []musicapi.Artist{},
		Albums:  []musicapi.Album{},
		Tracks:  []musicapi.Track{},
	}

	switch opts.Type {
	case "artist":
		artists, err := client.SearchArtists(ctx, opts.Query, opts.Limit)
		if err != nil {
			return nil, err
		}
		results.Artists = artists

	case "album":
		albums, err := client.SearchAlbums(ctx, opts.Query, opts.Limit)
		if err != nil {
			return nil, err
		}
		results.Albums = albums

	case "track":
		tracks, err := client.SearchTracks(ctx, opts.Query, opts.Limit)
		if err != nil {
			return nil, err
		}
		results.Tracks = tracks

	default: // "all" or empty
		searchResults, err := client.Search(ctx, opts.Query, opts.Limit)
		if err != nil {
			return nil, err
		}
		results.Artists = searchResults.Artists
		results.Albums = searchResults.Albums
		results.Tracks = searchResults.Tracks
	}

	return results, nil
}

// storeResults stores search results in the database
func (s *Service) storeResults(ctx context.Context, results *SearchResults) error {
	// Store artists
	for _, artist := range results.Artists {
		if err := s.storeArtist(ctx, artist); err != nil {
			log.Printf("Failed to store artist %s: %v", artist.Name, err)
		}
	}

	// Store albums
	for _, album := range results.Albums {
		if err := s.storeAlbum(ctx, album); err != nil {
			log.Printf("Failed to store album %s: %v", album.Title, err)
		}
	}

	// Store tracks
	for _, track := range results.Tracks {
		if err := s.storeTrack(ctx, track); err != nil {
			log.Printf("Failed to store track %s: %v", track.Title, err)
		}
	}

	return nil
}

// storeArtist stores an artist in the database if it doesn't exist
func (s *Service) storeArtist(ctx context.Context, artist musicapi.Artist) error {
	// Check if artist already exists by external_id and provider
	if artist.ExternalID != "" && artist.Provider != "" {
		var exists bool
		err := s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM artists WHERE external_id = $1 AND provider = $2)`,
			artist.ExternalID, artist.Provider).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check artist exists: %w", err)
		}

		if exists {
			// Update existing artist with latest data
			genresJSON, err := json.Marshal(artist.Genres)
			if err != nil {
				return fmt.Errorf("marshal genres: %w", err)
			}

			_, err = s.db.ExecContext(ctx, `
				UPDATE artists
				SET biography = $1,
				    image_url = $2,
				    genres = $3::jsonb,
				    popularity = $4,
				    external_url = $5,
				    updated_at = $6
				WHERE external_id = $7 AND provider = $8`,
				artist.Biography,
				artist.ImageURL,
				string(genresJSON),
				artist.Popularity,
				artist.ExternalURL,
				time.Now().UTC(),
				artist.ExternalID,
				artist.Provider,
			)

			if err != nil {
				log.Printf("Failed to update artist %s: %v", artist.Name, err)
			}
			return nil
		}
	}

	// Insert new artist with all fields
	genresJSON, err := json.Marshal(artist.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO artists (name, biography, image_url, external_id, provider, genres, popularity, external_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $9)
		ON CONFLICT (name) DO UPDATE SET
			biography = EXCLUDED.biography,
			image_url = EXCLUDED.image_url,
			external_id = EXCLUDED.external_id,
			provider = EXCLUDED.provider,
			genres = EXCLUDED.genres,
			popularity = EXCLUDED.popularity,
			external_url = EXCLUDED.external_url,
			updated_at = EXCLUDED.updated_at`,
		artist.Name,
		artist.Biography,
		artist.ImageURL,
		artist.ExternalID,
		artist.Provider,
		string(genresJSON),
		artist.Popularity,
		artist.ExternalURL,
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("insert artist: %w", err)
	}

	log.Printf("Stored artist: %s (from %s)", artist.Name, artist.Provider)
	return nil
}

// storeAlbum stores an album in the database if it doesn't exist
func (s *Service) storeAlbum(ctx context.Context, album musicapi.Album) error {
	// Catalog persistence requires user context; noop for provider result caching.
	return nil
}

// storeTrack stores a track in the database if it doesn't exist
func (s *Service) storeTrack(ctx context.Context, track musicapi.Track) error {
	// Catalog persistence requires user context; noop for provider result caching.
	return nil
}

// ImportAlbum fetches album details from a provider. Without user context the
// album is not persisted, but the fetch can be used to validate connectivity.
func (s *Service) ImportAlbum(ctx context.Context, albumID string, provider musicapi.MusicProvider) error {
	client, err := s.clientForProvider(provider)
	if err != nil {
		return err
	}

	log.Printf("ImportAlbum: validating album=%s provider=%s", albumID, provider)

	if _, _, err := client.GetAlbum(ctx, albumID); err != nil {
		return fmt.Errorf("fetch album: %w", err)
	}

	log.Printf("ImportAlbum: validation succeeded album=%s provider=%s", albumID, provider)

	return nil
}

// ImportAlbumForUser fetches a full album and stores it (and its tracks) for the authenticated user.
// Returns the database album ID and any error encountered.
func (s *Service) ImportAlbumForUser(ctx context.Context, token string, albumID string, provider musicapi.MusicProvider) (int64, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, store.ErrUnauthorized
	}
	if s.store == nil {
		log.Println("ImportAlbumForUser: store not configured")
		return 0, errors.New("store not configured")
	}

	userID, err := s.store.UserIDByToken(ctx, token)
	if err != nil {
		return 0, err
	}

	client, err := s.clientForProvider(provider)
	if err != nil {
		return 0, err
	}

	album, tracks, err := client.GetAlbum(ctx, albumID)
	if err != nil {
		return 0, fmt.Errorf("fetch album: %w", err)
	}

	log.Printf("ImportAlbumForUser: fetched album=%s provider=%s tracks=%d user=%d", album.Title, provider, len(tracks), userID)

	storedAlbumID, err := s.storeAlbumForUser(ctx, userID, *album, tracks)
	if err != nil {
		log.Printf("ImportAlbumForUser: failed storing album user=%d album=%s: %v", userID, album.Title, err)
		return 0, err
	}

	for _, track := range tracks {
		if err := s.storeTrackForUser(ctx, storedAlbumID, *album, track); err != nil {
			log.Printf("Failed to store track %s: %v", track.Title, err)
		}
	}

	log.Printf("Imported album: %s by %s with %d tracks for user %d (database ID: %d)", album.Title, album.Artist, len(tracks), userID, storedAlbumID)
	return storedAlbumID, nil
}

// Helper functions

func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullIfZero(i int) sql.NullInt32 {
	if i == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(i), Valid: true}
}

func (s *Service) clientForProvider(provider musicapi.MusicProvider) (musicapi.MusicAPIClient, error) {
	switch provider {
	case musicapi.ProviderSpotify:
		if s.spotifyClient == nil {
			return nil, fmt.Errorf("provider %s not configured", provider)
		}
		return s.spotifyClient, nil
	case musicapi.ProviderAppleMusic:
		if s.appleMusicClient == nil {
			return nil, fmt.Errorf("provider %s not configured", provider)
		}
		return s.appleMusicClient, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *Service) storeAlbumForUser(ctx context.Context, userID int64, album musicapi.Album, tracks []musicapi.Track) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id %d", userID)
	}

	var albumID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM albums
		WHERE user_id = $1 AND artist = $2 AND title = $3
	`, userID, album.Artist, album.Title).Scan(&albumID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup album: %w", err)
	}

	trackTitlesJSON, err := json.Marshal(extractTrackTitles(tracks))
	if err != nil {
		return 0, fmt.Errorf("marshal track titles: %w", err)
	}
	genresJSON, err := json.Marshal(extractGenres(album))
	if err != nil {
		return 0, fmt.Errorf("marshal genres: %w", err)
	}
	releaseYear := resolveReleaseYear(album)

	if albumID != 0 {
		log.Printf("storeAlbumForUser: updating existing album id=%d user=%d title=%q", albumID, userID, album.Title)
		if _, err := s.db.ExecContext(ctx, `
			UPDATE albums
			SET tracks = $1::jsonb,
			    genres = $2::jsonb,
			    release_year = $3
			WHERE id = $4
		`, string(trackTitlesJSON), string(genresJSON), releaseYear, albumID); err != nil {
			log.Printf("Failed to update album metadata (id=%d): %v", albumID, err)
		}
		return albumID, nil
	}

	const defaultRating = 3
	log.Printf("storeAlbumForUser: inserting album user=%d title=%q rating=%d", userID, album.Title, defaultRating)
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO albums (user_id, artist, title, release_year, tracks, genres, rating)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
		RETURNING id
	`, userID, album.Artist, album.Title, releaseYear, string(trackTitlesJSON), string(genresJSON), defaultRating).Scan(&albumID)
	if err != nil {
		return 0, fmt.Errorf("insert album: %w", err)
	}

	return albumID, nil
}

func (s *Service) storeTrackForUser(ctx context.Context, albumID int64, album musicapi.Album, track musicapi.Track) error {
	title := strings.TrimSpace(track.Title)
	if title == "" {
		return nil
	}
	artist := strings.TrimSpace(track.Artist)
	if artist == "" {
		artist = strings.TrimSpace(album.Artist)
	}
	if artist == "" {
		artist = album.Artist
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM songs
			WHERE album_id = $1 AND title = $2 AND artist = $3
		)
	`, albumID, title, artist).Scan(&exists); err != nil {
		return fmt.Errorf("check track exists: %w", err)
	}
	if exists {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO songs (title, artist, album_id, duration, track_num)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, title, artist, albumID, nullIfZero(track.Duration), nullIfZero(track.TrackNumber))
	if err != nil {
		return fmt.Errorf("insert track: %w", err)
	}

	return nil
}

func extractTrackTitles(tracks []musicapi.Track) []string {
	var titles []string
	for _, track := range tracks {
		title := strings.TrimSpace(track.Title)
		if title != "" {
			titles = append(titles, title)
		}
	}
	return titles
}

func extractGenres(album musicapi.Album) []string {
	if album.Genre == "" {
		return []string{}
	}
	parts := strings.Split(album.Genre, ",")
	var genres []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			genres = append(genres, trimmed)
		}
	}
	return genres
}

func resolveReleaseYear(album musicapi.Album) int {
	if album.ReleaseYear > 0 {
		return album.ReleaseYear
	}
	if year := parseYear(album.ReleaseDate); year > 0 {
		return year
	}
	return 1970
}

func parseYear(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil || year <= 0 {
		return 0
	}
	return year
}

// GetArtistWithAlbums fetches full artist details and all their albums from Spotify
func (s *Service) GetArtistWithAlbums(ctx context.Context, artistID string) (*musicapi.Artist, []musicapi.Album, error) {
	if s.spotifyClient == nil {
		return nil, nil, fmt.Errorf("spotify client not available")
	}

	// Type assert to get the concrete type with the GetArtistAlbums method
	spotifyClient, ok := s.spotifyClient.(*musicapi.SpotifyClient)
	if !ok {
		return nil, nil, fmt.Errorf("spotify client type assertion failed")
	}

	// Get artist details
	artist, err := spotifyClient.GetArtist(ctx, artistID)
	if err != nil {
		return nil, nil, fmt.Errorf("get artist: %w", err)
	}

	// Get all albums for this artist
	albums, err := spotifyClient.GetArtistAlbums(ctx, artistID)
	if err != nil {
		return nil, nil, fmt.Errorf("get artist albums: %w", err)
	}

	return artist, albums, nil
}

// GetAlbumWithTracks fetches full album details with all tracks from Spotify
func (s *Service) GetAlbumWithTracks(ctx context.Context, albumID string) (*musicapi.Album, []musicapi.Track, error) {
	if s.spotifyClient == nil {
		return nil, nil, fmt.Errorf("spotify client not available")
	}

	album, tracks, err := s.spotifyClient.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, nil, fmt.Errorf("get album: %w", err)
	}

	return album, tracks, nil
}

// GetAllArtists retrieves all artists from the database
func (s *Service) GetAllArtists(ctx context.Context) ([]musicapi.Artist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(external_id, '') as external_id,
			name,
			COALESCE(provider, '') as provider,
			COALESCE(image_url, '') as image_url,
			COALESCE(biography, '') as biography,
			COALESCE(genres, '[]'::jsonb) as genres,
			COALESCE(popularity, 0) as popularity,
			COALESCE(external_url, '') as external_url
		FROM artists
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query artists: %w", err)
	}
	defer rows.Close()

	var artists []musicapi.Artist
	for rows.Next() {
		var artist musicapi.Artist
		var genresJSON string

		err := rows.Scan(
			&artist.ExternalID,
			&artist.Name,
			&artist.Provider,
			&artist.ImageURL,
			&artist.Biography,
			&genresJSON,
			&artist.Popularity,
			&artist.ExternalURL,
		)
		if err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}

		// Parse genres JSON
		if err := json.Unmarshal([]byte(genresJSON), &artist.Genres); err != nil {
			log.Printf("Failed to parse genres for artist %s: %v", artist.Name, err)
			artist.Genres = []string{}
		}

		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return artists, nil
}

// SaveArtist stores an artist in the database (public wrapper for storeArtist)
func (s *Service) SaveArtist(ctx context.Context, artist musicapi.Artist) error {
	return s.storeArtist(ctx, artist)
}
