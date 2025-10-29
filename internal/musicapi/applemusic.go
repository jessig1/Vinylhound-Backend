package musicapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AppleMusicClient implements the MusicAPIClient interface for Apple Music
type AppleMusicClient struct {
	keyID      string
	teamID     string
	privateKey *ecdsa.PrivateKey
	httpClient *http.Client
	token      string
	tokenTime  time.Time
}

// NewAppleMusicClient creates a new Apple Music API client
func NewAppleMusicClient(keyID, teamID, privateKeyPEM string) (*AppleMusicClient, error) {
	// Parse the private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the key")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &AppleMusicClient{
		keyID:      keyID,
		teamID:     teamID,
		privateKey: privateKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Apple Music API response structures
type appleMusicSearchResponse struct {
	Results struct {
		Artists *appleMusicArtistsResults `json:"artists,omitempty"`
		Albums  *appleMusicAlbumsResults  `json:"albums,omitempty"`
		Songs   *appleMusicSongsResults   `json:"songs,omitempty"`
	} `json:"results"`
}

type appleMusicArtistsResults struct {
	Data []appleMusicArtist `json:"data"`
}

type appleMusicAlbumsResults struct {
	Data []appleMusicAlbum `json:"data"`
}

type appleMusicSongsResults struct {
	Data []appleMusicSong `json:"data"`
}

type appleMusicArtist struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Attributes appleMusicArtistAttributes `json:"attributes"`
}

type appleMusicArtistAttributes struct {
	Name      string   `json:"name"`
	GenreNames []string `json:"genreNames"`
	URL       string   `json:"url"`
}

type appleMusicAlbum struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Attributes appleMusicAlbumAttributes `json:"attributes"`
}

type appleMusicAlbumAttributes struct {
	Name         string                `json:"name"`
	ArtistName   string                `json:"artistName"`
	ReleaseDate  string                `json:"releaseDate"`
	GenreNames   []string              `json:"genreNames"`
	TrackCount   int                   `json:"trackCount"`
	Artwork      appleMusicArtwork     `json:"artwork"`
	URL          string                `json:"url"`
}

type appleMusicSong struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Attributes appleMusicSongAttributes `json:"attributes"`
}

type appleMusicSongAttributes struct {
	Name          string            `json:"name"`
	ArtistName    string            `json:"artistName"`
	AlbumName     string            `json:"albumName"`
	DurationInMillis int            `json:"durationInMillis"`
	TrackNumber   int               `json:"trackNumber"`
	DiscNumber    int               `json:"discNumber"`
	ISRC          string            `json:"isrc"`
	GenreNames    []string          `json:"genreNames"`
	Previews      []appleMusicPreview `json:"previews"`
	URL           string            `json:"url"`
}

type appleMusicArtwork struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type appleMusicPreview struct {
	URL string `json:"url"`
}

// generateToken creates a JWT token for Apple Music API authentication
func (c *AppleMusicClient) generateToken() (string, error) {
	// Check if we have a valid token (tokens last 6 months but we'll refresh every 12 hours)
	if c.token != "" && time.Since(c.tokenTime) < 12*time.Hour {
		return c.token, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.teamID,
		"iat": now.Unix(),
		"exp": now.Add(6 * 30 * 24 * time.Hour).Unix(), // 6 months
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = c.keyID

	tokenString, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	c.token = tokenString
	c.tokenTime = now

	return tokenString, nil
}

// doRequest performs an authenticated request to Apple Music API
func (c *AppleMusicClient) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	token, err := c.generateToken()
	if err != nil {
		return err
	}

	apiURL := "https://api.music.apple.com/v1/" + endpoint
	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Music-User-Token", "") // Optional: for user-specific data

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apple music api error: %s - %s", resp.Status, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// SearchArtists searches for artists on Apple Music
func (c *AppleMusicClient) SearchArtists(ctx context.Context, query string, limit int) ([]Artist, error) {
	params := url.Values{
		"term":  []string{query},
		"types": []string{"artists"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result appleMusicSearchResponse
	if err := c.doRequest(ctx, "catalog/us/search", params, &result); err != nil {
		return nil, err
	}

	if result.Results.Artists == nil {
		return []Artist{}, nil
	}

	artists := make([]Artist, 0, len(result.Results.Artists.Data))
	for _, aa := range result.Results.Artists.Data {
		artists = append(artists, c.convertArtist(aa))
	}

	return artists, nil
}

// SearchAlbums searches for albums on Apple Music
func (c *AppleMusicClient) SearchAlbums(ctx context.Context, query string, limit int) ([]Album, error) {
	params := url.Values{
		"term":  []string{query},
		"types": []string{"albums"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result appleMusicSearchResponse
	if err := c.doRequest(ctx, "catalog/us/search", params, &result); err != nil {
		return nil, err
	}

	if result.Results.Albums == nil {
		return []Album{}, nil
	}

	albums := make([]Album, 0, len(result.Results.Albums.Data))
	for _, aa := range result.Results.Albums.Data {
		albums = append(albums, c.convertAlbum(aa))
	}

	return albums, nil
}

// SearchTracks searches for tracks on Apple Music
func (c *AppleMusicClient) SearchTracks(ctx context.Context, query string, limit int) ([]Track, error) {
	params := url.Values{
		"term":  []string{query},
		"types": []string{"songs"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result appleMusicSearchResponse
	if err := c.doRequest(ctx, "catalog/us/search", params, &result); err != nil {
		return nil, err
	}

	if result.Results.Songs == nil {
		return []Track{}, nil
	}

	tracks := make([]Track, 0, len(result.Results.Songs.Data))
	for _, as := range result.Results.Songs.Data {
		tracks = append(tracks, c.convertTrack(as))
	}

	return tracks, nil
}

// Search performs a combined search across all types
func (c *AppleMusicClient) Search(ctx context.Context, query string, limit int) (*SearchResults, error) {
	params := url.Values{
		"term":  []string{query},
		"types": []string{"artists,albums,songs"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result appleMusicSearchResponse
	if err := c.doRequest(ctx, "catalog/us/search", params, &result); err != nil {
		return nil, err
	}

	searchResults := &SearchResults{
		Artists: []Artist{},
		Albums:  []Album{},
		Tracks:  []Track{},
	}

	if result.Results.Artists != nil {
		for _, aa := range result.Results.Artists.Data {
			searchResults.Artists = append(searchResults.Artists, c.convertArtist(aa))
		}
	}

	if result.Results.Albums != nil {
		for _, aa := range result.Results.Albums.Data {
			searchResults.Albums = append(searchResults.Albums, c.convertAlbum(aa))
		}
	}

	if result.Results.Songs != nil {
		for _, as := range result.Results.Songs.Data {
			searchResults.Tracks = append(searchResults.Tracks, c.convertTrack(as))
		}
	}

	return searchResults, nil
}

// GetArtist retrieves full artist details by ID
func (c *AppleMusicClient) GetArtist(ctx context.Context, artistID string) (*Artist, error) {
	type artistResponse struct {
		Data []appleMusicArtist `json:"data"`
	}

	var result artistResponse
	if err := c.doRequest(ctx, "catalog/us/artists/"+artistID, nil, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("artist not found")
	}

	artist := c.convertArtist(result.Data[0])
	return &artist, nil
}

// GetAlbum retrieves full album details including tracks by ID
func (c *AppleMusicClient) GetAlbum(ctx context.Context, albumID string) (*Album, []Track, error) {
	type albumResponse struct {
		Data []struct {
			appleMusicAlbum
			Relationships struct {
				Tracks struct {
					Data []appleMusicSong `json:"data"`
				} `json:"tracks"`
			} `json:"relationships"`
		} `json:"data"`
	}

	params := url.Values{
		"include": []string{"tracks"},
	}

	var result albumResponse
	if err := c.doRequest(ctx, "catalog/us/albums/"+albumID, params, &result); err != nil {
		return nil, nil, err
	}

	if len(result.Data) == 0 {
		return nil, nil, fmt.Errorf("album not found")
	}

	album := c.convertAlbum(result.Data[0].appleMusicAlbum)

	tracks := []Track{}
	for _, as := range result.Data[0].Relationships.Tracks.Data {
		tracks = append(tracks, c.convertTrack(as))
	}

	return &album, tracks, nil
}

// GetTrack retrieves full track details by ID
func (c *AppleMusicClient) GetTrack(ctx context.Context, trackID string) (*Track, error) {
	type trackResponse struct {
		Data []appleMusicSong `json:"data"`
	}

	var result trackResponse
	if err := c.doRequest(ctx, "catalog/us/songs/"+trackID, nil, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("track not found")
	}

	track := c.convertTrack(result.Data[0])
	return &track, nil
}

// Helper functions to convert Apple Music types to common types

func (c *AppleMusicClient) convertArtist(aa appleMusicArtist) Artist {
	return Artist{
		ExternalID:  aa.ID,
		Name:        aa.Attributes.Name,
		Provider:    ProviderAppleMusic,
		Genres:      aa.Attributes.GenreNames,
		ExternalURL: aa.Attributes.URL,
	}
}

func (c *AppleMusicClient) convertAlbum(aa appleMusicAlbum) Album {
	// Replace placeholders in artwork URL
	coverURL := strings.ReplaceAll(aa.Attributes.Artwork.URL, "{w}", "600")
	coverURL = strings.ReplaceAll(coverURL, "{h}", "600")

	releaseYear := 0
	if len(aa.Attributes.ReleaseDate) >= 4 {
		fmt.Sscanf(aa.Attributes.ReleaseDate[:4], "%d", &releaseYear)
	}

	genre := ""
	if len(aa.Attributes.GenreNames) > 0 {
		genre = aa.Attributes.GenreNames[0]
	}

	return Album{
		ExternalID:  aa.ID,
		Title:       aa.Attributes.Name,
		Artist:      aa.Attributes.ArtistName,
		Provider:    ProviderAppleMusic,
		ReleaseYear: releaseYear,
		ReleaseDate: aa.Attributes.ReleaseDate,
		Genre:       genre,
		CoverURL:    coverURL,
		TrackCount:  aa.Attributes.TrackCount,
		ExternalURL: aa.Attributes.URL,
	}
}

func (c *AppleMusicClient) convertTrack(as appleMusicSong) Track {
	previewURL := ""
	if len(as.Attributes.Previews) > 0 {
		previewURL = as.Attributes.Previews[0].URL
	}

	return Track{
		ExternalID:  as.ID,
		Title:       as.Attributes.Name,
		Artist:      as.Attributes.ArtistName,
		Album:       as.Attributes.AlbumName,
		Provider:    ProviderAppleMusic,
		Duration:    as.Attributes.DurationInMillis / 1000, // Convert ms to seconds
		TrackNumber: as.Attributes.TrackNumber,
		DiscNumber:  as.Attributes.DiscNumber,
		ISRC:        as.Attributes.ISRC,
		ExternalURL: as.Attributes.URL,
		PreviewURL:  previewURL,
	}
}
