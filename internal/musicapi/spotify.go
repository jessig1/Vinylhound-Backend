package musicapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SpotifyClient implements the MusicAPIClient interface for Spotify
type SpotifyClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
	mu           sync.RWMutex
}

// NewSpotifyClient creates a new Spotify API client
func NewSpotifyClient(clientID, clientSecret string) *SpotifyClient {
	return &SpotifyClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Spotify API response structures
type spotifySearchResponse struct {
	Artists *spotifyArtistsPage `json:"artists,omitempty"`
	Albums  *spotifyAlbumsPage  `json:"albums,omitempty"`
	Tracks  *spotifyTracksPage  `json:"tracks,omitempty"`
}

type spotifyArtistsPage struct {
	Items []spotifyArtist `json:"items"`
}

type spotifyAlbumsPage struct {
	Items []spotifyAlbum `json:"items"`
}

type spotifyTracksPage struct {
	Items []spotifyTrack `json:"items"`
}

type spotifyArtist struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Genres     []string            `json:"genres"`
	Popularity int                 `json:"popularity"`
	Images     []spotifyImage      `json:"images"`
	ExternalURLs spotifyExternalURLs `json:"external_urls"`
}

type spotifyAlbum struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Artists      []spotifySimpleArtist `json:"artists"`
	ReleaseDate  string                `json:"release_date"`
	TotalTracks  int                   `json:"total_tracks"`
	Images       []spotifyImage        `json:"images"`
	ExternalURLs spotifyExternalURLs   `json:"external_urls"`
	Tracks       *spotifyTracksPage    `json:"tracks,omitempty"`
}

type spotifyTrack struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Artists      []spotifySimpleArtist `json:"artists"`
	Album        *spotifySimpleAlbum   `json:"album,omitempty"`
	Duration     int                   `json:"duration_ms"`
	TrackNumber  int                   `json:"track_number"`
	DiscNumber   int                   `json:"disc_number"`
	ISRC         string                `json:"external_ids.isrc,omitempty"`
	PreviewURL   string                `json:"preview_url,omitempty"`
	ExternalURLs spotifyExternalURLs   `json:"external_urls"`
}

type spotifySimpleArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type spotifySimpleAlbum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type spotifyImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type spotifyExternalURLs struct {
	Spotify string `json:"spotify"`
}

type spotifyTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// authenticate obtains an access token from Spotify
func (c *SpotifyClient) authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if token is still valid
	if time.Now().Before(c.tokenExpiry) {
		return nil
	}

	// Prepare authentication request
	authString := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+authString)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify auth failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp spotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// doRequest performs an authenticated request to Spotify API
func (c *SpotifyClient) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	apiURL := "https://api.spotify.com/v1/" + endpoint
	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify api error: %s - %s", resp.Status, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// SearchArtists searches for artists on Spotify
func (c *SpotifyClient) SearchArtists(ctx context.Context, query string, limit int) ([]Artist, error) {
	params := url.Values{
		"q":     []string{query},
		"type":  []string{"artist"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result spotifySearchResponse
	if err := c.doRequest(ctx, "search", params, &result); err != nil {
		return nil, err
	}

	if result.Artists == nil {
		return []Artist{}, nil
	}

	artists := make([]Artist, 0, len(result.Artists.Items))
	for _, sa := range result.Artists.Items {
		artists = append(artists, c.convertArtist(sa))
	}

	return artists, nil
}

// SearchAlbums searches for albums on Spotify
func (c *SpotifyClient) SearchAlbums(ctx context.Context, query string, limit int) ([]Album, error) {
	params := url.Values{
		"q":     []string{query},
		"type":  []string{"album"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result spotifySearchResponse
	if err := c.doRequest(ctx, "search", params, &result); err != nil {
		return nil, err
	}

	if result.Albums == nil {
		return []Album{}, nil
	}

	albums := make([]Album, 0, len(result.Albums.Items))
	for _, sa := range result.Albums.Items {
		albums = append(albums, c.convertAlbum(sa))
	}

	return albums, nil
}

// SearchTracks searches for tracks on Spotify
func (c *SpotifyClient) SearchTracks(ctx context.Context, query string, limit int) ([]Track, error) {
	params := url.Values{
		"q":     []string{query},
		"type":  []string{"track"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result spotifySearchResponse
	if err := c.doRequest(ctx, "search", params, &result); err != nil {
		return nil, err
	}

	if result.Tracks == nil {
		return []Track{}, nil
	}

	tracks := make([]Track, 0, len(result.Tracks.Items))
	for _, st := range result.Tracks.Items {
		tracks = append(tracks, c.convertTrack(st))
	}

	return tracks, nil
}

// Search performs a combined search across all types
func (c *SpotifyClient) Search(ctx context.Context, query string, limit int) (*SearchResults, error) {
	params := url.Values{
		"q":     []string{query},
		"type":  []string{"artist,album,track"},
		"limit": []string{fmt.Sprintf("%d", limit)},
	}

	var result spotifySearchResponse
	if err := c.doRequest(ctx, "search", params, &result); err != nil {
		return nil, err
	}

	searchResults := &SearchResults{
		Artists: []Artist{},
		Albums:  []Album{},
		Tracks:  []Track{},
	}

	if result.Artists != nil {
		for _, sa := range result.Artists.Items {
			searchResults.Artists = append(searchResults.Artists, c.convertArtist(sa))
		}
	}

	if result.Albums != nil {
		for _, sa := range result.Albums.Items {
			searchResults.Albums = append(searchResults.Albums, c.convertAlbum(sa))
		}
	}

	if result.Tracks != nil {
		for _, st := range result.Tracks.Items {
			searchResults.Tracks = append(searchResults.Tracks, c.convertTrack(st))
		}
	}

	return searchResults, nil
}

// GetArtist retrieves full artist details by ID
func (c *SpotifyClient) GetArtist(ctx context.Context, artistID string) (*Artist, error) {
	var sa spotifyArtist
	if err := c.doRequest(ctx, "artists/"+artistID, nil, &sa); err != nil {
		return nil, err
	}

	artist := c.convertArtist(sa)
	return &artist, nil
}

// GetArtistAlbums retrieves all albums by an artist
func (c *SpotifyClient) GetArtistAlbums(ctx context.Context, artistID string) ([]Album, error) {
	params := url.Values{}
	params.Set("include_groups", "album,single")
	params.Set("limit", "50")

	var response struct {
		Items []spotifyAlbum `json:"items"`
		Next  string         `json:"next"`
	}

	if err := c.doRequest(ctx, "artists/"+artistID+"/albums", params, &response); err != nil {
		return nil, err
	}

	albums := make([]Album, 0, len(response.Items))
	for _, sa := range response.Items {
		albums = append(albums, c.convertAlbum(sa))
	}

	return albums, nil
}

// GetAlbum retrieves full album details including tracks by ID
func (c *SpotifyClient) GetAlbum(ctx context.Context, albumID string) (*Album, []Track, error) {
	var sa spotifyAlbum
	if err := c.doRequest(ctx, "albums/"+albumID, nil, &sa); err != nil {
		return nil, nil, err
	}

	album := c.convertAlbum(sa)

	tracks := []Track{}
	if sa.Tracks != nil {
		for _, st := range sa.Tracks.Items {
			// Add album info to track
			st.Album = &spotifySimpleAlbum{
				ID:   sa.ID,
				Name: sa.Name,
			}
			tracks = append(tracks, c.convertTrack(st))
		}
	}

	return &album, tracks, nil
}

// GetTrack retrieves full track details by ID
func (c *SpotifyClient) GetTrack(ctx context.Context, trackID string) (*Track, error) {
	var st spotifyTrack
	if err := c.doRequest(ctx, "tracks/"+trackID, nil, &st); err != nil {
		return nil, err
	}

	track := c.convertTrack(st)
	return &track, nil
}

// Helper functions to convert Spotify types to common types

func (c *SpotifyClient) convertArtist(sa spotifyArtist) Artist {
	imageURL := ""
	if len(sa.Images) > 0 {
		imageURL = sa.Images[0].URL
	}

	return Artist{
		ExternalID:  sa.ID,
		Name:        sa.Name,
		Provider:    ProviderSpotify,
		ImageURL:    imageURL,
		Genres:      sa.Genres,
		Popularity:  sa.Popularity,
		ExternalURL: sa.ExternalURLs.Spotify,
	}
}

func (c *SpotifyClient) convertAlbum(sa spotifyAlbum) Album {
	artistName := ""
	artistID := ""
	if len(sa.Artists) > 0 {
		artistName = sa.Artists[0].Name
		artistID = sa.Artists[0].ID
	}

	coverURL := ""
	if len(sa.Images) > 0 {
		coverURL = sa.Images[0].URL
	}

	releaseYear := 0
	if len(sa.ReleaseDate) >= 4 {
		fmt.Sscanf(sa.ReleaseDate[:4], "%d", &releaseYear)
	}

	return Album{
		ExternalID:  sa.ID,
		Title:       sa.Name,
		Artist:      artistName,
		ArtistID:    artistID,
		Provider:    ProviderSpotify,
		ReleaseYear: releaseYear,
		ReleaseDate: sa.ReleaseDate,
		CoverURL:    coverURL,
		TrackCount:  sa.TotalTracks,
		ExternalURL: sa.ExternalURLs.Spotify,
	}
}

func (c *SpotifyClient) convertTrack(st spotifyTrack) Track {
	artistName := ""
	artistID := ""
	if len(st.Artists) > 0 {
		artistName = st.Artists[0].Name
		artistID = st.Artists[0].ID
	}

	albumName := ""
	albumID := ""
	if st.Album != nil {
		albumName = st.Album.Name
		albumID = st.Album.ID
	}

	return Track{
		ExternalID:  st.ID,
		Title:       st.Name,
		Artist:      artistName,
		ArtistID:    artistID,
		Album:       albumName,
		AlbumID:     albumID,
		Provider:    ProviderSpotify,
		Duration:    st.Duration / 1000, // Convert ms to seconds
		TrackNumber: st.TrackNumber,
		DiscNumber:  st.DiscNumber,
		ISRC:        st.ISRC,
		ExternalURL: st.ExternalURLs.Spotify,
		PreviewURL:  st.PreviewURL,
	}
}
