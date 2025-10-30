package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vinylhound/internal/store"
	"vinylhound/shared/go/models"
)

// handleFavorites handles legacy favorites routes (not yet implemented).
func (s *Server) handleFavorites(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "authorization required"})
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
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	}
}

// handleCheckFavorite handles GET for checking if an item is favorited
// GET /api/v1/favorites/check?song_id=123 or ?album_id=456
func (s *Server) handleCheckFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "authorization required"})
		return
	}

	s.checkFavorite(w, r, token)
}

// handleFavoritesPlaylist handles GET for the favorites playlist
// GET /api/v1/favorites/playlist
func (s *Server) handleFavoritesPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "authorization required"})
		return
	}

	s.getFavoritesPlaylist(w, r, token)
}

func (s *Server) addFavorite(w http.ResponseWriter, r *http.Request, token string) {
	var req models.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "favorites service not implemented"})
}

func (s *Server) removeFavorite(w http.ResponseWriter, r *http.Request, token string) {
	var req models.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "favorites service not implemented"})
}

func (s *Server) listFavorites(w http.ResponseWriter, r *http.Request, token string) {
	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "favorites service not implemented"})
}

func (s *Server) checkFavorite(w http.ResponseWriter, r *http.Request, token string) {
	songIDStr := r.URL.Query().Get("song_id")
	albumIDStr := r.URL.Query().Get("album_id")

	var songID, albumID *int64

	if songIDStr != "" {
		id, err := strconv.ParseInt(songIDStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid song_id"})
			return
		}
		songID = &id
	}

	if albumIDStr != "" {
		id, err := strconv.ParseInt(albumIDStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid album_id"})
			return
		}
		albumID = &id
	}

	if songID == nil && albumID == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "song_id or album_id required"})
		return
	}

	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "favorites service not implemented"})
}

func (s *Server) getFavoritesPlaylist(w http.ResponseWriter, r *http.Request, token string) {
	writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "favorites service not implemented"})
}

type favoriteTrackView struct {
	TrackID     int64     `json:"track_id"`
	FavoritedAt time.Time `json:"favorited_at"`
}

type favoriteTrackEnvelope struct {
	Track favoriteTrackView `json:"track"`
}

type favoriteTracksResponse struct {
	Tracks []favoriteTrackView `json:"tracks"`
}

func (s *Server) handleFavoriteTracks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	favorites, err := s.favorites.ListTrackFavorites(r.Context(), token)
	if err != nil {
		status, message := mapFavoritesError(err)
		writeJSON(w, status, errorResponse{Error: message})
		return
	}

	resp := favoriteTracksResponse{
		Tracks: make([]favoriteTrackView, 0, len(favorites)),
	}
	for _, fav := range favorites {
		if fav == nil || fav.SongID == nil {
			continue
		}
		resp.Tracks = append(resp.Tracks, favoriteTrackView{
			TrackID:     *fav.SongID,
			FavoritedAt: fav.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleFavoriteTrack(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/me/favorites/tracks/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/api/me/favorites/tracks/")
	}

	if path == "" || path == r.URL.Path {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "track id required"})
		return
	}

	trackID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || trackID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "trackId must be a positive integer"})
		return
	}

	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		fav, created, err := s.favorites.FavoriteTrack(r.Context(), token, trackID)
		if err != nil {
			status, message := mapFavoritesError(err)
			writeJSON(w, status, errorResponse{Error: message})
			return
		}

		if !created {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if fav == nil || fav.SongID == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "favorite created without track reference"})
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/api/v1/me/favorites/tracks/%d", trackID))
		writeJSON(w, http.StatusCreated, favoriteTrackEnvelope{
			Track: favoriteTrackView{
				TrackID:     *fav.SongID,
				FavoritedAt: fav.CreatedAt,
			},
		})
	case http.MethodDelete:
		if err := s.favorites.UnfavoriteTrack(r.Context(), token, trackID); err != nil {
			status, message := mapFavoritesError(err)
			writeJSON(w, status, errorResponse{Error: message})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodPut, http.MethodDelete))
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	}
}

func mapFavoritesError(err error) (int, string) {
	switch {
	case errors.Is(err, store.ErrUnauthorized):
		return http.StatusUnauthorized, "authorization required"
	case errors.Is(err, store.ErrFavoriteNotFound):
		return http.StatusNotFound, "favorite not found"
	case errors.Is(err, store.ErrInvalidFavoriteType):
		return http.StatusBadRequest, "invalid favorite type"
	default:
		return http.StatusInternalServerError, err.Error()
	}
}
