package musicapi

import (
	"context"
	"time"
)

// MusicProvider represents a music streaming service
type MusicProvider string

const (
	ProviderSpotify    MusicProvider = "spotify"
	ProviderAppleMusic MusicProvider = "apple_music"
)

// Artist represents an artist from an external music service
type Artist struct {
	ExternalID   string        `json:"external_id"`
	Name         string        `json:"name"`
	Provider     MusicProvider `json:"provider"`
	ImageURL     string        `json:"image_url,omitempty"`
	Biography    string        `json:"biography,omitempty"`
	Genres       []string      `json:"genres,omitempty"`
	Popularity   int           `json:"popularity,omitempty"`
	ExternalURL  string        `json:"external_url,omitempty"`
}

// Album represents an album from an external music service
type Album struct {
	ExternalID   string        `json:"external_id"`
	Title        string        `json:"title"`
	Artist       string        `json:"artist"`
	ArtistID     string        `json:"artist_id,omitempty"`
	Provider     MusicProvider `json:"provider"`
	ReleaseYear  int           `json:"release_year,omitempty"`
	ReleaseDate  string        `json:"release_date,omitempty"`
	Genre        string        `json:"genre,omitempty"`
	CoverURL     string        `json:"cover_url,omitempty"`
	TrackCount   int           `json:"track_count,omitempty"`
	ExternalURL  string        `json:"external_url,omitempty"`
}

// Track represents a track/song from an external music service
type Track struct {
	ExternalID   string        `json:"external_id"`
	Title        string        `json:"title"`
	Artist       string        `json:"artist"`
	ArtistID     string        `json:"artist_id,omitempty"`
	Album        string        `json:"album,omitempty"`
	AlbumID      string        `json:"album_id,omitempty"`
	Provider     MusicProvider `json:"provider"`
	Duration     int           `json:"duration"` // in seconds
	TrackNumber  int           `json:"track_number,omitempty"`
	DiscNumber   int           `json:"disc_number,omitempty"`
	ISRC         string        `json:"isrc,omitempty"`
	ExternalURL  string        `json:"external_url,omitempty"`
	PreviewURL   string        `json:"preview_url,omitempty"`
}

// SearchResults contains results from a music API search
type SearchResults struct {
	Artists []Artist `json:"artists"`
	Albums  []Album  `json:"albums"`
	Tracks  []Track  `json:"tracks"`
}

// MusicAPIClient defines the interface for music streaming service clients
type MusicAPIClient interface {
	// SearchArtists searches for artists by name
	SearchArtists(ctx context.Context, query string, limit int) ([]Artist, error)

	// SearchAlbums searches for albums by title or artist
	SearchAlbums(ctx context.Context, query string, limit int) ([]Album, error)

	// SearchTracks searches for tracks by title or artist
	SearchTracks(ctx context.Context, query string, limit int) ([]Track, error)

	// Search performs a combined search across all types
	Search(ctx context.Context, query string, limit int) (*SearchResults, error)

	// GetArtist retrieves full artist details by ID
	GetArtist(ctx context.Context, artistID string) (*Artist, error)

	// GetAlbum retrieves full album details including tracks by ID
	GetAlbum(ctx context.Context, albumID string) (*Album, []Track, error)

	// GetTrack retrieves full track details by ID
	GetTrack(ctx context.Context, trackID string) (*Track, error)
}

// Config holds configuration for music API clients
type Config struct {
	// Spotify credentials
	SpotifyClientID     string
	SpotifyClientSecret string

	// Apple Music credentials
	AppleMusicKeyID     string
	AppleMusicTeamID    string
	AppleMusicPrivateKey string

	// Rate limiting
	RequestTimeout time.Duration
	MaxRetries     int
}
