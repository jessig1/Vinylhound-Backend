package searchservice

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"vinylhound/internal/musicapi"
	"vinylhound/internal/store"
)

// Service provides unified search across multiple music providers and stores results
type Service struct {
	db              *sql.DB
	spotifyClient   musicapi.MusicAPIClient
	appleMusicClient musicapi.MusicAPIClient
	store           *store.Store
}

// NewService creates a new search service
func NewService(db *sql.DB, spotifyClient, appleMusicClient musicapi.MusicAPIClient, st *store.Store) *Service {
	return &Service{
		db:              db,
		spotifyClient:   spotifyClient,
		appleMusicClient: appleMusicClient,
		store:           st,
	}
}

// SearchOptions defines search parameters
type SearchOptions struct {
	Query       string
	Type        string // "artist", "album", "track", or "all"
	Provider    string // "spotify", "apple_music", or "all"
	Limit       int
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
	// Check if artist already exists
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM artists WHERE name = $1)`,
		artist.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check artist exists: %w", err)
	}

	if exists {
		return nil // Already exists
	}

	// Insert new artist
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO artists (name, biography, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (name) DO NOTHING`,
		artist.Name,
		artist.Biography,
		artist.ImageURL,
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
	// Check if album already exists
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM albums WHERE title = $1 AND artist = $2)`,
		album.Title, album.Artist).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check album exists: %w", err)
	}

	if exists {
		return nil // Already exists
	}

	// Insert new album
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO albums (title, artist, release_year, genre, cover_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT DO NOTHING`,
		album.Title,
		album.Artist,
		nullIfZero(album.ReleaseYear),
		nullIfEmpty(album.Genre),
		nullIfEmpty(album.CoverURL),
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("insert album: %w", err)
	}

	log.Printf("Stored album: %s by %s (from %s)", album.Title, album.Artist, album.Provider)
	return nil
}

// storeTrack stores a track in the database if it doesn't exist
func (s *Service) storeTrack(ctx context.Context, track musicapi.Track) error {
	// Get or create album first
	var albumID sql.NullInt64
	if track.Album != "" {
		var aid int64
		err := s.db.QueryRowContext(ctx,
			`SELECT id FROM albums WHERE title = $1 AND artist = $2`,
			track.Album, track.Artist).Scan(&aid)

		if err == nil {
			albumID = sql.NullInt64{Int64: aid, Valid: true}
		} else if err == sql.ErrNoRows {
			// Album doesn't exist, create it
			err = s.db.QueryRowContext(ctx, `
				INSERT INTO albums (title, artist, created_at, updated_at)
				VALUES ($1, $2, $3, $3)
				RETURNING id`,
				track.Album,
				track.Artist,
				time.Now().UTC(),
			).Scan(&aid)

			if err == nil {
				albumID = sql.NullInt64{Int64: aid, Valid: true}
				log.Printf("Created album: %s by %s", track.Album, track.Artist)
			} else {
				log.Printf("Failed to create album for track: %v", err)
			}
		}
	}

	// Check if track already exists
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM songs WHERE title = $1 AND artist = $2)`,
		track.Title, track.Artist).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check track exists: %w", err)
	}

	if exists {
		return nil // Already exists
	}

	// Insert new track
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO songs (title, artist, album_id, duration, track_num, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT DO NOTHING`,
		track.Title,
		track.Artist,
		albumID,
		nullIfZero(track.Duration),
		nullIfZero(track.TrackNumber),
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("insert track: %w", err)
	}

	log.Printf("Stored track: %s by %s (from %s)", track.Title, track.Artist, track.Provider)
	return nil
}

// ImportAlbum fetches full album details from a provider and stores it with all tracks
func (s *Service) ImportAlbum(ctx context.Context, albumID string, provider musicapi.MusicProvider) error {
	var client musicapi.MusicAPIClient

	switch provider {
	case musicapi.ProviderSpotify:
		client = s.spotifyClient
	case musicapi.ProviderAppleMusic:
		client = s.appleMusicClient
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	if client == nil {
		return fmt.Errorf("provider %s not configured", provider)
	}

	// Fetch album details
	album, tracks, err := client.GetAlbum(ctx, albumID)
	if err != nil {
		return fmt.Errorf("fetch album: %w", err)
	}

	// Store album
	if err := s.storeAlbum(ctx, *album); err != nil {
		return fmt.Errorf("store album: %w", err)
	}

	// Store all tracks
	for _, track := range tracks {
		if err := s.storeTrack(ctx, track); err != nil {
			log.Printf("Failed to store track %s: %v", track.Title, err)
		}
	}

	log.Printf("Imported album: %s by %s with %d tracks", album.Title, album.Artist, len(tracks))
	return nil
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
