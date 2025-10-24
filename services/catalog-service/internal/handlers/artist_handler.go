package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/catalog-service/internal/service"

	"github.com/gorilla/mux"
)

// ArtistHandler handles HTTP requests for artist operations
type ArtistHandler struct {
	artistService *service.ArtistService
}

// NewArtistHandler creates a new artist handler
func NewArtistHandler(artistService *service.ArtistService) *ArtistHandler {
	return &ArtistHandler{artistService: artistService}
}

// ListArtists handles listing artists with optional filtering
func (h *ArtistHandler) ListArtists(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	artists, err := h.artistService.ListArtists(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"artists": artists})
}

// GetArtist handles getting a specific artist
func (h *ArtistHandler) GetArtist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid artist ID", http.StatusBadRequest)
		return
	}

	artist, err := h.artistService.GetArtist(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(artist)
}
