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

	if err := s.searchService.ImportAlbumForUser(r.Context(), token, req.AlbumID, provider); err != nil {
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

	log.Printf("ImportAlbum: completed import album=%s provider=%s", req.AlbumID, provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Album imported successfully",
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
