package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"vinylhound/internal/store"
	"vinylhound/shared/go/models"
)

// FavoritesService coordinates favorites-related operations.
type FavoritesService interface {
	AddFavorite(token string, songID *int64, albumID *int64) (*models.Favorite, error)
	RemoveFavorite(token string, songID *int64, albumID *int64) error
	ListFavorites(token string) ([]*models.Favorite, error)
	IsFavorite(token string, songID *int64, albumID *int64) (bool, error)
	GetFavoritesPlaylist(token string) (*models.Playlist, error)
}

// handleFavorites handles GET (list) and POST (add) and DELETE (remove) for favorites
func (s *Server) handleFavorites(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listFavorites(w, r, token)
	case http.MethodPost:
		s.addFavorite(w, r, token)
	case http.MethodDelete:
		s.removeFavorite(w, r, token)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCheckFavorite handles GET for checking if an item is favorited
// GET /api/v1/favorites/check?song_id=123 or ?album_id=456
func (s *Server) handleCheckFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	s.checkFavorite(w, r, token)
}

// handleFavoritesPlaylist handles GET for the favorites playlist
// GET /api/v1/favorites/playlist
func (s *Server) handleFavoritesPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	s.getFavoritesPlaylist(w, r, token)
}

func (s *Server) addFavorite(w http.ResponseWriter, r *http.Request, token string) {
	var req models.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Note: We need a FavoritesService - for now we'll need to add this to the store
	// This is a simplified version that would need proper service layer integration
	http.Error(w, "Not implemented - requires favorites service", http.StatusNotImplemented)
}

func (s *Server) removeFavorite(w http.ResponseWriter, r *http.Request, token string) {
	var req models.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Note: We need a FavoritesService
	http.Error(w, "Not implemented - requires favorites service", http.StatusNotImplemented)
}

func (s *Server) listFavorites(w http.ResponseWriter, r *http.Request, token string) {
	// Note: We need a FavoritesService
	http.Error(w, "Not implemented - requires favorites service", http.StatusNotImplemented)
}

func (s *Server) checkFavorite(w http.ResponseWriter, r *http.Request, token string) {
	songIDStr := r.URL.Query().Get("song_id")
	albumIDStr := r.URL.Query().Get("album_id")

	var songID, albumID *int64

	if songIDStr != "" {
		id, err := strconv.ParseInt(songIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid song_id", http.StatusBadRequest)
			return
		}
		songID = &id
	}

	if albumIDStr != "" {
		id, err := strconv.ParseInt(albumIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid album_id", http.StatusBadRequest)
			return
		}
		albumID = &id
	}

	// Note: We need a FavoritesService
	http.Error(w, "Not implemented - requires favorites service", http.StatusNotImplemented)
}

func (s *Server) getFavoritesPlaylist(w http.ResponseWriter, r *http.Request, token string) {
	// Note: We need a FavoritesService
	http.Error(w, "Not implemented - requires favorites service", http.StatusNotImplemented)
}
