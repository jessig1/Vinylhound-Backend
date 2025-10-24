package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/catalog-service/internal/service"

	"github.com/gorilla/mux"
)

// SongHandler handles HTTP requests for song operations
type SongHandler struct {
	songService *service.SongService
}

// NewSongHandler creates a new song handler
func NewSongHandler(songService *service.SongService) *SongHandler {
	return &SongHandler{songService: songService}
}

// ListSongs handles listing songs with optional filtering
func (h *SongHandler) ListSongs(w http.ResponseWriter, r *http.Request) {
	var albumID int64
	var artist string

	if albumIDStr := r.URL.Query().Get("album_id"); albumIDStr != "" {
		if id, err := strconv.ParseInt(albumIDStr, 10, 64); err == nil {
			albumID = id
		}
	}

	artist = r.URL.Query().Get("artist")

	songs, err := h.songService.ListSongs(r.Context(), albumID, artist)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"songs": songs})
}

// GetSong handles getting a specific song
func (h *SongHandler) GetSong(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	song, err := h.songService.GetSong(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(song)
}
