package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"vinylhound/shared/go/models"
)

// extractToken extracts the bearer token from the Authorization header
func extractToken(r *http.Request) string {
	return parseBearerToken(r.Header.Get("Authorization"))
}

// handlePlaylists handles GET (list) and POST (create) for playlists
func (s *Server) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPlaylists(w, r)
	case http.MethodPost:
		s.createPlaylist(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePlaylist handles GET/PUT/DELETE for a specific playlist, and POST/DELETE for playlist songs
func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/playlists/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Playlist ID required", http.StatusBadRequest)
		return
	}

	playlistID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	// Check if this is a songs sub-resource
	if len(parts) >= 2 && parts[1] == "songs" {
		if len(parts) == 2 {
			// /playlists/{id}/songs
			if r.Method == http.MethodPost {
				s.addSongToPlaylist(w, r, playlistID)
				return
			}
		} else if len(parts) == 3 {
			// /playlists/{id}/songs/{songId}
			songID, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				http.Error(w, "Invalid song ID", http.StatusBadRequest)
				return
			}
			if r.Method == http.MethodDelete {
				s.removeSongFromPlaylist(w, r, playlistID, songID)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle playlist CRUD
	switch r.Method {
	case http.MethodGet:
		s.getPlaylist(w, r, playlistID)
	case http.MethodPut:
		s.updatePlaylist(w, r, playlistID)
	case http.MethodDelete:
		s.deletePlaylist(w, r, playlistID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listPlaylists(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	playlists, err := s.playlists.List(r.Context(), token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Playlists []*models.Playlist `json:"playlists"`
	}{Playlists: playlists})
}

func (s *Server) getPlaylist(w http.ResponseWriter, r *http.Request, id int64) {
	playlist, err := s.playlists.Get(r.Context(), id)
	if err != nil {
		if err.Error() == "playlist not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}

func (s *Server) createPlaylist(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title       string                `json:"title"`
		Description string                `json:"description"`
		IsPublic    bool                  `json:"isPublic"`
		Tags        []string              `json:"tags"`
		Songs       []models.PlaylistSong `json:"songs,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	playlist := &models.Playlist{
		Title:       req.Title,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		Tags:        req.Tags,
		Songs:       req.Songs, // Will be nil or empty slice
	}

	created, err := s.playlists.Create(r.Context(), token, playlist)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) updatePlaylist(w http.ResponseWriter, r *http.Request, id int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title       string                `json:"title"`
		Description string                `json:"description"`
		IsPublic    bool                  `json:"isPublic"`
		Tags        []string              `json:"tags"`
		Songs       []models.PlaylistSong `json:"songs,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	playlist := &models.Playlist{
		Title:       req.Title,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		Tags:        req.Tags,
		Songs:       req.Songs, // Will be nil if not provided in request
	}

	updated, err := s.playlists.Update(r.Context(), token, id, playlist)
	if err != nil {
		if err.Error() == "playlist not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) deletePlaylist(w http.ResponseWriter, r *http.Request, id int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	err := s.playlists.Delete(r.Context(), token, id)
	if err != nil {
		if err.Error() == "playlist not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addSongToPlaylist(w http.ResponseWriter, r *http.Request, playlistID int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		SongID int64 `json:"song_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.playlists.AddSong(r.Context(), token, playlistID, req.SongID)
	if err != nil {
		if err.Error() == "playlist not found" || err.Error() == "song not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch and return the updated playlist with songs
	playlist, err := s.playlists.Get(r.Context(), playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}

func (s *Server) removeSongFromPlaylist(w http.ResponseWriter, r *http.Request, playlistID, songID int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	err := s.playlists.RemoveSong(r.Context(), token, playlistID, songID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch and return the updated playlist with songs
	playlist, err := s.playlists.Get(r.Context(), playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}
