package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/catalog-service/internal/service"
	"vinylhound/shared/models"

	"github.com/gorilla/mux"
)

// AlbumHandler handles HTTP requests for album operations
type AlbumHandler struct {
	albumService *service.AlbumService
}

// NewAlbumHandler creates a new album handler
func NewAlbumHandler(albumService *service.AlbumService) *AlbumHandler {
	return &AlbumHandler{albumService: albumService}
}

// ListAlbums handles listing albums with optional filtering
func (h *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := models.AlbumFilter{
		Artist:     r.URL.Query().Get("artist"),
		Genre:      r.URL.Query().Get("genre"),
		SearchTerm: r.URL.Query().Get("search"),
	}

	// Parse year filters
	if yearFrom := r.URL.Query().Get("year_from"); yearFrom != "" {
		if year, err := strconv.Atoi(yearFrom); err == nil {
			filter.YearFrom = year
		}
	}
	if yearTo := r.URL.Query().Get("year_to"); yearTo != "" {
		if year, err := strconv.Atoi(yearTo); err == nil {
			filter.YearTo = year
		}
	}

	// Parse pagination
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	albums, err := h.albumService.ListAlbums(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string][]*models.Album{"albums": albums})
}

// GetAlbum handles getting a specific album
func (h *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	album, err := h.albumService.GetAlbum(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(album)
}

// CreateAlbum handles creating a new album
func (h *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var album models.Album
	if err := json.NewDecoder(r.Body).Decode(&album); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if album.Title == "" || album.Artist == "" {
		http.Error(w, "Title and artist are required", http.StatusBadRequest)
		return
	}

	createdAlbum, err := h.albumService.CreateAlbum(r.Context(), &album)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdAlbum)
}

// UpdateAlbum handles updating an existing album
func (h *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	var album models.Album
	if err := json.NewDecoder(r.Body).Decode(&album); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedAlbum, err := h.albumService.UpdateAlbum(r.Context(), id, &album)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedAlbum)
}

// DeleteAlbum handles deleting an album
func (h *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	if err := h.albumService.DeleteAlbum(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
