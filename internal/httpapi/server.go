package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"vinylhound/internal/app/artists"
	"vinylhound/internal/app/songs"
	"vinylhound/internal/musicapi"
	"vinylhound/internal/searchservice"
	"vinylhound/internal/store"
	"vinylhound/shared/go/models"
)

// UserService captures the user-facing operations needed by the HTTP handlers.
type UserService interface {
	Signup(ctx context.Context, username, password string, content []string) error
	Authenticate(ctx context.Context, username, password string) (string, error)
	Content(ctx context.Context, token string) ([]string, error)
	UpdateContent(ctx context.Context, token string, content []string) error
}

// ArtistService describes artist catalogue workflows.
type ArtistService interface {
	List(ctx context.Context, filter artists.Filter) ([]artists.Artist, error)
}

// AlbumService exposes album-specific workflows.
type AlbumService interface {
	Create(ctx context.Context, token string, album store.Album) (store.Album, error)
	ListByUser(ctx context.Context, token string) ([]store.Album, error)
	List(ctx context.Context, filter store.AlbumFilter) ([]store.Album, error)
	Get(ctx context.Context, id int64) (store.Album, error)
}

// SongService coordinates track-level operations.
type SongService interface {
	ListByAlbum(ctx context.Context, albumID int64) ([]songs.Song, error)
	Search(ctx context.Context, filter store.SongFilter) ([]store.Song, error)
	Get(ctx context.Context, id int64) (store.Song, error)
}

// RatingsService describes preference-related workflows.
type RatingsService interface {
	Upsert(ctx context.Context, token string, albumID int64, rating *int, favorited bool) error
	ListByUser(ctx context.Context, token string) ([]store.AlbumPreference, error)
}

// PlaylistService coordinates playlist-related operations.
type PlaylistService interface {
	List(ctx context.Context, token string) ([]*models.Playlist, error)
	Get(ctx context.Context, id int64) (*models.Playlist, error)
	Create(ctx context.Context, token string, playlist *models.Playlist) (*models.Playlist, error)
	Update(ctx context.Context, token string, id int64, playlist *models.Playlist) (*models.Playlist, error)
	Delete(ctx context.Context, token string, id int64) error
	AddSong(ctx context.Context, token string, playlistID int64, songID int64) error
	RemoveSong(ctx context.Context, token string, playlistID int64, songID int64) error
}

// FavoritesService coordinates favoriting workflows.
type FavoritesService interface {
	FavoriteTrack(ctx context.Context, token string, trackID int64) (*models.Favorite, bool, error)
	UnfavoriteTrack(ctx context.Context, token string, trackID int64) error
	ListTrackFavorites(ctx context.Context, token string) ([]*models.Favorite, error)
}

// SearchService provides unified search across music providers.
type SearchService interface {
	Search(ctx context.Context, opts searchservice.SearchOptions) (*searchservice.SearchResults, error)
	ImportAlbumForUser(ctx context.Context, token string, albumID string, provider musicapi.MusicProvider) (int64, error)
	ImportAlbum(ctx context.Context, albumID string, provider musicapi.MusicProvider) error
	GetArtistWithAlbums(ctx context.Context, artistID string) (*musicapi.Artist, []musicapi.Album, error)
	GetAlbumWithTracks(ctx context.Context, albumID string) (*musicapi.Album, []musicapi.Track, error)
	GetAllArtists(ctx context.Context) ([]musicapi.Artist, error)
	SaveArtist(ctx context.Context, artist musicapi.Artist) error
}

// PlaceService coordinates place-related operations (venues and retailers)
type PlaceService interface {
	CreateVenue(ctx context.Context, token string, venue *models.Venue) (*models.Venue, error)
	ListVenues(ctx context.Context, token string) ([]*models.Venue, error)
	GetVenue(ctx context.Context, id int64) (*models.Venue, error)
	UpdateVenue(ctx context.Context, token string, id int64, venue *models.Venue) (*models.Venue, error)
	DeleteVenue(ctx context.Context, token string, id int64) error
	CreateRetailer(ctx context.Context, token string, retailer *models.Retailer) (*models.Retailer, error)
	ListRetailers(ctx context.Context, token string) ([]*models.Retailer, error)
	GetRetailer(ctx context.Context, id int64) (*models.Retailer, error)
	UpdateRetailer(ctx context.Context, token string, id int64, retailer *models.Retailer) (*models.Retailer, error)
	DeleteRetailer(ctx context.Context, token string, id int64) error
}

// ConcertService coordinates concert-related operations
type ConcertService interface {
	Create(ctx context.Context, token string, concert *models.Concert) (*models.Concert, error)
	List(ctx context.Context, token string) ([]*models.ConcertWithDetails, error)
	Get(ctx context.Context, id int64) (*models.ConcertWithDetails, error)
	Update(ctx context.Context, token string, id int64, concert *models.Concert) (*models.Concert, error)
	Delete(ctx context.Context, token string, id int64) error
	ListUpcoming(ctx context.Context, token string) ([]*models.ConcertWithDetails, error)
	ListByVenue(ctx context.Context, venueID int64) ([]*models.ConcertWithDetails, error)
	ListByArtist(ctx context.Context, token string, artistName string) ([]*models.ConcertWithDetails, error)
	MarkAttended(ctx context.Context, token string, concertID int64, rating *int) error
}

// CollectionService coordinates album collection operations (wishlist and owned)
type CollectionService interface {
	Add(ctx context.Context, token string, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	List(ctx context.Context, token string, filter models.CollectionFilter) ([]*models.AlbumCollectionWithDetails, error)
	Get(ctx context.Context, id int64) (*models.AlbumCollectionWithDetails, error)
	Update(ctx context.Context, token string, id int64, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	Remove(ctx context.Context, token string, id int64) error
	Move(ctx context.Context, token string, id int64, targetType models.CollectionType) error
	GetStats(ctx context.Context, token string) (*models.CollectionStats, error)
}

// Server wires HTTP handlers to the underlying services.
type Server struct {
	users         UserService
	artists       ArtistService
	albums        AlbumService
	songs         SongService
	ratings       RatingsService
	playlists     PlaylistService
	favorites     FavoritesService
	searchService SearchService
	places        PlaceService
	concerts      ConcertService
	collections   CollectionService
}

// New configures a Server with the given Store implementation.
func New(
	users UserService,
	artists ArtistService,
	albums AlbumService,
	songs SongService,
	ratings RatingsService,
	playlists PlaylistService,
	favorites FavoritesService,
	searchService SearchService,
	places PlaceService,
	concerts ConcertService,
	collections CollectionService,
) *Server {
	return &Server{
		users:         users,
		artists:       artists,
		albums:        albums,
		songs:         songs,
		ratings:       ratings,
		playlists:     playlists,
		favorites:     favorites,
		searchService: searchService,
		places:        places,
		concerts:      concerts,
		collections:   collections,
	}
}

// Routes exposes the HTTP handlers for account and content management.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API v1 routes (standardized)
	mux.HandleFunc("/api/v1/auth/signup", s.handleSignup)
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/users/profile", s.handleContent) // me/content -> users/profile
	mux.HandleFunc("/api/v1/me/albums", s.handleAlbums)
	mux.HandleFunc("/api/v1/me/albums/preferences", s.handleAlbumPreferences)
	mux.HandleFunc("/api/v1/me/albums/", s.handleAlbumPreference)
	mux.HandleFunc("/api/v1/albums", s.handleAlbumsList)
	mux.HandleFunc("/api/v1/albums/", s.handleAlbum) // Changed from /api/album

	// Playlist routes
	mux.HandleFunc("/api/v1/playlists", s.handlePlaylists)
	mux.HandleFunc("/api/v1/playlists/", s.handlePlaylist)

	// Song routes
	mux.HandleFunc("/api/v1/songs", s.handleSongs)
	mux.HandleFunc("/api/v1/songs/", s.handleSong)

	// Favorite track routes
	mux.HandleFunc("/api/v1/me/favorites/tracks", s.handleFavoriteTracks)
	mux.HandleFunc("/api/v1/me/favorites/tracks/", s.handleFavoriteTrack)
	// Legacy favorites track routes (pre-v1 prefix)
	mux.HandleFunc("/api/me/favorites/tracks", s.handleFavoriteTracks)
	mux.HandleFunc("/api/me/favorites/tracks/", s.handleFavoriteTrack)

	// Favorites routes
	mux.HandleFunc("/api/v1/favorites", s.handleFavorites)
	mux.HandleFunc("/api/v1/favorites/check", s.handleCheckFavorite)
	mux.HandleFunc("/api/v1/favorites/playlist", s.handleFavoritesPlaylist)

	// Search routes
	mux.HandleFunc("/api/v1/search", s.handleSearch)
	mux.HandleFunc("/api/v1/import/album", s.handleImportAlbum)
	mux.HandleFunc("/api/v1/providers", s.handleProviders)
	mux.HandleFunc("/api/v1/artist", s.handleGetArtist)
	mux.HandleFunc("/api/v1/album/details", s.handleGetAlbumDetails)
	mux.HandleFunc("/api/v1/artists", s.handleArtists)

	// Venue routes
	mux.HandleFunc("POST /api/v1/venues", s.handleCreateVenue)
	mux.HandleFunc("GET /api/v1/venues", s.handleListVenues)
	mux.HandleFunc("GET /api/v1/venues/{id}", s.handleGetVenue)
	mux.HandleFunc("PUT /api/v1/venues/{id}", s.handleUpdateVenue)
	mux.HandleFunc("DELETE /api/v1/venues/{id}", s.handleDeleteVenue)

	// Retailer routes
	mux.HandleFunc("POST /api/v1/retailers", s.handleCreateRetailer)
	mux.HandleFunc("GET /api/v1/retailers", s.handleListRetailers)
	mux.HandleFunc("GET /api/v1/retailers/{id}", s.handleGetRetailer)
	mux.HandleFunc("PUT /api/v1/retailers/{id}", s.handleUpdateRetailer)
	mux.HandleFunc("DELETE /api/v1/retailers/{id}", s.handleDeleteRetailer)

	// Concert routes
	mux.HandleFunc("POST /api/v1/concerts", s.handleCreateConcert)
	mux.HandleFunc("GET /api/v1/concerts", s.handleListConcerts)
	mux.HandleFunc("GET /api/v1/concerts/{id}", s.handleGetConcert)
	mux.HandleFunc("PUT /api/v1/concerts/{id}", s.handleUpdateConcert)
	mux.HandleFunc("DELETE /api/v1/concerts/{id}", s.handleDeleteConcert)
	mux.HandleFunc("POST /api/v1/concerts/{id}/attend", s.handleMarkConcertAttended)

	// Collection routes (album wishlist and owned)
	mux.HandleFunc("/api/v1/collections", s.handleCollections)
	mux.HandleFunc("/api/v1/collections/", s.handleCollection)
	mux.HandleFunc("/api/v1/collections/stats", s.handleCollectionStats)

	// Legacy routes (for backward compatibility) - TODO: Remove after frontend migration
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

	if err := s.users.Signup(r.Context(), req.Username, req.Password, req.Content); err != nil {
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

	token, err := s.users.Authenticate(r.Context(), req.Username, req.Password)
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
		content, err := s.users.Content(r.Context(), token)
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
		if err := s.users.UpdateContent(r.Context(), token, body.Content); err != nil {
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
		albums, err := s.albums.ListByUser(r.Context(), token)
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

		created, err := s.albums.Create(r.Context(), token, album)
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

	prefs, err := s.ratings.ListByUser(r.Context(), token)
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

	// Handle both /api/v1/me/albums/ and /api/me/albums/ paths
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/me/albums/")
	if trimmed == r.URL.Path {
		// Path didn't match v1, try legacy path
		trimmed = strings.TrimPrefix(r.URL.Path, "/api/me/albums/")
	}

	if trimmed == "" || trimmed == r.URL.Path {
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

		if err := s.ratings.Upsert(r.Context(), token, albumID, req.Rating, req.Favorited); err != nil {
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
		if err := s.ratings.Upsert(r.Context(), token, albumID, nil, false); err != nil {
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

	albums, err := s.albums.List(r.Context(), filter)
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

	// Extract ID from URL path: /api/v1/albums/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/albums/")

	// If TrimPrefix didn't change the path, try legacy /api/album?id=X format
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/api/album")
		if path == "" || path == r.URL.Path {
			// Not a valid path, try query parameter
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id parameter"})
				return
			}
			path = idStr
		}
	}

	// Parse the ID from the path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing id parameter"})
		return
	}

	idStr := parts[0]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid id parameter"})
		return
	}

	album, err := s.albums.Get(r.Context(), id)
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
