package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"vinylhound/internal/musicapi"
	"vinylhound/internal/searchservice"
	"vinylhound/internal/store"
)

// handleSearch performs unified search across music providers
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query        string `json:"query"`
		Type         string `json:"type"`     // "artist", "album", "track", or "all"
		Provider     string `json:"provider"` // "spotify", "apple_music", or "all"
		Limit        int    `json:"limit"`
		StoreResults bool   `json:"store_results"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	opts := searchservice.SearchOptions{
		Query:        req.Query,
		Type:         req.Type,
		Provider:     req.Provider,
		Limit:        req.Limit,
		StoreResults: req.StoreResults,
	}

	results, err := s.searchService.Search(r.Context(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleImportAlbum imports a complete album from a provider
func (s *Server) handleImportAlbum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		log.Println("ImportAlbum: missing bearer token")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req struct {
		AlbumID  string `json:"album_id"`
		Provider string `json:"provider"` // "spotify" or "apple_music"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ImportAlbum: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AlbumID == "" {
		log.Println("ImportAlbum: missing album_id")
		http.Error(w, "Album ID is required", http.StatusBadRequest)
		return
	}

	if req.Provider == "" {
		log.Println("ImportAlbum: missing provider")
		http.Error(w, "Provider is required", http.StatusBadRequest)
		return
	}

	var provider musicapi.MusicProvider
	switch req.Provider {
	case "spotify":
		provider = musicapi.ProviderSpotify
	case "apple_music":
		provider = musicapi.ProviderAppleMusic
	default:
		log.Printf("ImportAlbum: invalid provider %q", req.Provider)
		http.Error(w, "Invalid provider. Must be 'spotify' or 'apple_music'", http.StatusBadRequest)
		return
	}

	log.Printf("ImportAlbum: attempting import album=%s provider=%s", req.AlbumID, provider)

	dbAlbumID, err := s.searchService.ImportAlbumForUser(r.Context(), token, req.AlbumID, provider)
	if err != nil {
		if errors.Is(err, store.ErrUnauthorized) {
			log.Printf("ImportAlbum: unauthorized import attempt for album=%s provider=%s", req.AlbumID, provider)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("ImportAlbum: ERROR - failed importing album=%s provider=%s: %v", req.AlbumID, provider, err)

		// Return the actual error message to help with debugging
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("Failed to import album: %v", err)})
		return
	}

	log.Printf("ImportAlbum: completed import album=%s provider=%s database_id=%d", req.AlbumID, provider, dbAlbumID)

	// Retrieve the stored album to return full details
	album, err := s.albums.Get(r.Context(), dbAlbumID)
	if err != nil {
		log.Printf("ImportAlbum: ERROR - failed to retrieve stored album id=%d: %v", dbAlbumID, err)
		// Return success but with just the ID since the album was imported
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Album imported successfully",
			"album": map[string]interface{}{
				"id": dbAlbumID,
			},
		})
		return
	}

	log.Printf("ImportAlbum: returning album details id=%d title=%q artist=%q", album.ID, album.Title, album.Artist)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Album imported successfully",
		"album":   album,
	})
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// handleProviders returns list of available music providers
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := []map[string]interface{}{
		{
			"id":   "spotify",
			"name": "Spotify",
		},
		{
			"id":   "apple_music",
			"name": "Apple Music",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
	})
}

// handleGetArtist retrieves full artist details from Spotify
func (s *Server) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artistID := r.URL.Query().Get("id")
	if artistID == "" {
		http.Error(w, "Artist ID is required", http.StatusBadRequest)
		return
	}

	artist, albums, err := s.searchService.GetArtistWithAlbums(r.Context(), artistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"artist": artist,
		"albums": albums,
	})
}

// handleArtists handles both GET (list artists) and POST (save artist)
func (s *Server) handleArtists(w http.ResponseWriter, r *http.Request) {
	log.Printf("[handleArtists] Method: %s, Path: %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		s.handleListArtists(w, r)
	case http.MethodPost:
		s.handleSaveArtist(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListArtists retrieves all artists from the database
func (s *Server) handleListArtists(w http.ResponseWriter, r *http.Request) {
	artists, err := s.searchService.GetAllArtists(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"artists": artists,
	})
}

// handleSaveArtist saves an artist to the database
func (s *Server) handleSaveArtist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExternalID  string   `json:"external_id"`
		Name        string   `json:"name"`
		Provider    string   `json:"provider"`
		ImageURL    string   `json:"image_url"`
		Biography   string   `json:"biography"`
		Genres      []string `json:"genres"`
		Popularity  int      `json:"popularity"`
		ExternalURL string   `json:"external_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Artist name is required", http.StatusBadRequest)
		return
	}

	artist := musicapi.Artist{
		ExternalID:  req.ExternalID,
		Name:        req.Name,
		Provider:    musicapi.MusicProvider(req.Provider),
		ImageURL:    req.ImageURL,
		Biography:   req.Biography,
		Genres:      req.Genres,
		Popularity:  req.Popularity,
		ExternalURL: req.ExternalURL,
	}

	if err := s.searchService.SaveArtist(r.Context(), artist); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Artist saved successfully",
	})
}

// handleGetAlbumDetails retrieves full album details including all tracks from Spotify
func (s *Server) handleGetAlbumDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	albumID := r.URL.Query().Get("id")
	if albumID == "" {
		http.Error(w, "Album ID is required", http.StatusBadRequest)
		return
	}

	album, tracks, err := s.searchService.GetAlbumWithTracks(r.Context(), albumID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"album":  album,
		"tracks": tracks,
	})
}
