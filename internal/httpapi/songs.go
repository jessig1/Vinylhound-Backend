package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"vinylhound/internal/store"
)

// handleSongs handles GET (search/list) for songs
func (s *Server) handleSongs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	filter := store.SongFilter{
		Query:  query.Get("q"),
		Artist: query.Get("artist"),
		Album:  query.Get("album"),
	}

	// Handle album_id filter
	if albumIDStr := query.Get("album_id"); albumIDStr != "" {
		albumID, err := strconv.ParseInt(albumIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid album_id", http.StatusBadRequest)
			return
		}
		filter.AlbumID = &albumID
	}

	songs, err := s.songs.Search(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Songs []store.Song `json:"songs"`
	}{
		Songs: songs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSong handles GET for a specific song by ID
func (s *Server) handleSong(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/songs/")
	if path == "" {
		http.Error(w, "Song ID required", http.StatusBadRequest)
		return
	}

	songID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	song, err := s.songs.Get(r.Context(), songID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Song not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(song)
}
