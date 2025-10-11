package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"vinylhound/internal/store"
)

// ContentStore exposes the persistence operations needed by the HTTP layer.
type ContentStore interface {
	CreateUser(username, password string, content []string) error
	Authenticate(username, password string) (string, error)
	ContentByToken(token string) ([]string, error)
	UpdateContentByToken(token string, content []string) error
	CreateAlbum(token string, album store.Album) (store.Album, error)
	AlbumsByToken(token string) ([]store.Album, error)
	ListAlbums(filter store.AlbumFilter) ([]store.Album, error)
	AlbumByID(id int64) (store.Album, error)
	UpsertAlbumPreference(token string, albumID int64, rating *int, favorited bool) error
	AlbumPreferencesByToken(token string) ([]store.AlbumPreference, error)
}

// Server wires HTTP handlers to the backing store.
type Server struct {
	store ContentStore
}

// New configures a Server with the given Store implementation.
func New(store ContentStore) *Server {
	return &Server{store: store}
}

// Routes exposes the HTTP handlers for account and content management.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/signup", s.handleSignup)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/me/content", s.handleContent)
	mux.HandleFunc("/api/me/albums", s.handleAlbums)
	mux.HandleFunc("/api/me/albums/preferences", s.handleAlbumPreferences)
	mux.HandleFunc("/api/me/albums/", s.handleAlbumPreference)
	mux.HandleFunc("/api/albums", s.handleAlbumsList)
	mux.HandleFunc("/api/album", s.handleAlbum)
	return mux
}

type signupRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Content  []string `json:"content"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type albumRequest struct {
	Artist      string   `json:"artist"`
	Title       string   `json:"title"`
	ReleaseYear int      `json:"releaseYear"`
	Tracks      []string `json:"trackList"`
	Genres      []string `json:"genreList"`
	Rating      int      `json:"rating"`
}

type albumPreferenceRequest struct {
	Rating    *int `json:"rating"`
	Favorited bool `json:"favorited"`
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	if err := s.store.CreateUser(req.Username, req.Password, req.Content); err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "username already taken"})
		default:
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	token, err := s.store.Authenticate(req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, store.ErrInvalidCredentials) {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		content, err := s.store.ContentByToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			if !errors.Is(err, store.ErrUnauthorized) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Content []string `json:"content"`
		}{Content: content})
	case http.MethodPut:
		var body struct {
			Content []string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
			return
		}
		if err := s.store.UpdateContentByToken(token, body.Content); err != nil {
			status := http.StatusUnauthorized
			if !errors.Is(err, store.ErrUnauthorized) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlbums(w http.ResponseWriter, r *http.Request) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		albums, err := s.store.AlbumsByToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			if !errors.Is(err, store.ErrUnauthorized) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Albums []store.Album `json:"albums"`
		}{Albums: albums})
	case http.MethodPost:
		var req albumRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
			return
		}

		album := store.Album{
			Artist:      req.Artist,
			Title:       req.Title,
			ReleaseYear: req.ReleaseYear,
			Tracks:      req.Tracks,
			Genres:      req.Genres,
			Rating:      req.Rating,
		}

		created, err := s.store.CreateAlbum(token, album)
		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, store.ErrUnauthorized):
				status = http.StatusUnauthorized
			case errors.Is(err, store.ErrInvalidAlbum):
				status = http.StatusBadRequest
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlbumPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	prefs, err := s.store.AlbumPreferencesByToken(token)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, store.ErrUnauthorized) {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Preferences []store.AlbumPreference `json:"preferences"`
	}{Preferences: prefs})
}

func (s *Server) handleAlbumPreference(w http.ResponseWriter, r *http.Request) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/me/albums/")
	if trimmed == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "preference" {
		http.NotFound(w, r)
		return
	}

	albumID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid album id"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req albumPreferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
			return
		}

		if err := s.store.UpsertAlbumPreference(token, albumID, req.Rating, req.Favorited); err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, store.ErrUnauthorized):
				status = http.StatusUnauthorized
			case errors.Is(err, store.ErrInvalidAlbum):
				status = http.StatusBadRequest
			case errors.Is(err, store.ErrAlbumNotFound):
				status = http.StatusNotFound
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.store.UpsertAlbumPreference(token, albumID, nil, false); err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, store.ErrUnauthorized):
				status = http.StatusUnauthorized
			case errors.Is(err, store.ErrAlbumNotFound):
				status = http.StatusNotFound
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlbumsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	filter := store.AlbumFilter{
		Artist: query.Get("artist"),
		Title:  query.Get("title"),
		Genre:  query.Get("genre"),
	}

	if yearStr := query.Get("year"); yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid year parameter"})
			return
		}
		filter.ReleaseYear = year
	}

	if ratingStr := query.Get("rating"); ratingStr != "" {
		rating, err := strconv.Atoi(ratingStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid rating parameter"})
			return
		}
		filter.Rating = rating
	}

	albums, err := s.store.ListAlbums(filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Albums []store.Album `json:"albums"`
	}{Albums: albums})
}

func (s *Server) handleAlbum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id parameter"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid id parameter"})
		return
	}

	album, err := s.store.AlbumByID(id)
	if err != nil {
		if errors.Is(err, store.ErrAlbumNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "album not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, album)
}

func parseBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
