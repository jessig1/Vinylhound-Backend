package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vinylhound/internal/app/artists"
	"vinylhound/internal/app/songs"
	"vinylhound/internal/musicapi"
	"vinylhound/internal/searchservice"
	"vinylhound/internal/store"
	"vinylhound/shared/go/models"
)

type stubUserService struct{}

func (stubUserService) Signup(context.Context, string, string, []string) error {
	return nil
}

func (stubUserService) Authenticate(context.Context, string, string) (string, error) {
	return "", nil
}

func (stubUserService) Content(context.Context, string) ([]string, error) {
	return nil, nil
}

func (stubUserService) UpdateContent(context.Context, string, []string) error {
	return nil
}

type stubAlbumService struct {
	albumsResponse []store.Album
	albumsErr      error

	createdAlbum store.Album
	createErr    error

	listAlbumsResponse []store.Album
	listAlbumsErr      error

	singleAlbum store.Album
	singleErr   error

	lastToken string
}

func (s *stubAlbumService) Create(ctx context.Context, token string, album store.Album) (store.Album, error) {
	s.lastToken = token
	s.createdAlbum = album
	if s.createErr != nil {
		return store.Album{}, s.createErr
	}
	return s.createdAlbum, nil
}

func (s *stubAlbumService) ListByUser(ctx context.Context, token string) ([]store.Album, error) {
	s.lastToken = token
	if s.albumsErr != nil {
		return nil, s.albumsErr
	}
	return s.albumsResponse, nil
}

func (s *stubAlbumService) List(ctx context.Context, filter store.AlbumFilter) ([]store.Album, error) {
	if s.listAlbumsErr != nil {
		return nil, s.listAlbumsErr
	}
	return s.listAlbumsResponse, nil
}

func (s *stubAlbumService) Get(ctx context.Context, id int64) (store.Album, error) {
	if s.singleErr != nil {
		return store.Album{}, s.singleErr
	}
	return s.singleAlbum, nil
}

type stubRatingsService struct {
	preferencesResponse []store.AlbumPreference
	preferencesErr      error

	upsertErr     error
	lastAlbumID   int64
	lastRating    *int
	lastFavorited bool

	lastToken string
}

func (s *stubRatingsService) Upsert(ctx context.Context, token string, albumID int64, rating *int, favorited bool) error {
	s.lastToken = token
	s.lastAlbumID = albumID
	s.lastFavorited = favorited
	if rating != nil {
		val := *rating
		s.lastRating = &val
	} else {
		s.lastRating = nil
	}
	if s.upsertErr != nil {
		return s.upsertErr
	}
	return nil
}

func (s *stubRatingsService) ListByUser(ctx context.Context, token string) ([]store.AlbumPreference, error) {
	s.lastToken = token
	if s.preferencesErr != nil {
		return nil, s.preferencesErr
	}
	return s.preferencesResponse, nil
}

type stubPlaylistService struct{}

func (stubPlaylistService) List(context.Context, string) ([]*models.Playlist, error) { return nil, nil }
func (stubPlaylistService) Get(context.Context, int64) (*models.Playlist, error)     { return nil, nil }
func (stubPlaylistService) Create(context.Context, string, *models.Playlist) (*models.Playlist, error) {
	return nil, nil
}
func (stubPlaylistService) Update(context.Context, string, int64, *models.Playlist) (*models.Playlist, error) {
	return nil, nil
}
func (stubPlaylistService) Delete(context.Context, string, int64) error { return nil }
func (stubPlaylistService) AddSong(context.Context, string, int64, int64) error {
	return nil
}
func (stubPlaylistService) RemoveSong(context.Context, string, int64, int64) error {
	return nil
}

type stubFavoritesService struct {
	favoriteTrackResponse *models.Favorite
	favoriteTrackCreated  bool
	favoriteTrackErr      error

	unfavoriteErr error

	listResponse []*models.Favorite
	listErr      error

	lastFavoriteToken   string
	lastFavoriteTrackID int64

	lastUnfavoriteToken   string
	lastUnfavoriteTrackID int64

	lastListToken string
}

func (s *stubFavoritesService) FavoriteTrack(ctx context.Context, token string, trackID int64) (*models.Favorite, bool, error) {
	s.lastFavoriteToken = token
	s.lastFavoriteTrackID = trackID
	if s.favoriteTrackErr != nil {
		return nil, false, s.favoriteTrackErr
	}
	if s.favoriteTrackCreated {
		return s.favoriteTrackResponse, true, nil
	}
	return nil, false, nil
}

func (s *stubFavoritesService) UnfavoriteTrack(ctx context.Context, token string, trackID int64) error {
	s.lastUnfavoriteToken = token
	s.lastUnfavoriteTrackID = trackID
	if s.unfavoriteErr != nil {
		return s.unfavoriteErr
	}
	return nil
}

func (s *stubFavoritesService) ListTrackFavorites(ctx context.Context, token string) ([]*models.Favorite, error) {
	s.lastListToken = token
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listResponse, nil
}

type noopArtistService struct{}

func (noopArtistService) List(context.Context, artists.Filter) ([]artists.Artist, error) {
	return nil, nil
}

type noopSongService struct{}

func (noopSongService) ListByAlbum(context.Context, int64) ([]songs.Song, error) {
	return nil, nil
}

func (noopSongService) Search(context.Context, store.SongFilter) ([]store.Song, error) {
	return nil, nil
}

func (noopSongService) Get(context.Context, int64) (store.Song, error) {
	return store.Song{}, nil
}

type noopSearchService struct{}

func (noopSearchService) Search(context.Context, searchservice.SearchOptions) (*searchservice.SearchResults, error) {
	return nil, nil
}

func (noopSearchService) ImportAlbum(context.Context, string, musicapi.MusicProvider) error {
	return nil
}

func (noopSearchService) GetArtistWithAlbums(context.Context, string) (*musicapi.Artist, []musicapi.Album, error) {
	return nil, nil, nil
}

func (noopSearchService) GetAlbumWithTracks(context.Context, string) (*musicapi.Album, []musicapi.Track, error) {
	return nil, nil, nil
}

func newTestServer(t *testing.T, album *stubAlbumService, ratings *stubRatingsService, favorites *stubFavoritesService) *Server {
	t.Helper()
	if album == nil {
		album = &stubAlbumService{}
	}
	if ratings == nil {
		ratings = &stubRatingsService{}
	}
	if favorites == nil {
		favorites = &stubFavoritesService{}
	}
	return New(
		&stubUserService{},
		noopArtistService{},
		album,
		noopSongService{},
		ratings,
		stubPlaylistService{},
		favorites,
		noopSearchService{},
	)
}

func TestHandleAlbumsGetSuccess(t *testing.T) {
	albumStub := &stubAlbumService{
		albumsResponse: []store.Album{
			{ID: 1, Artist: "Artist", Title: "Title", ReleaseYear: 2000, Rating: 4},
		},
	}
	server := newTestServer(t, albumStub, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/me/albums", nil)
	req.Header.Set("Authorization", "Bearer token-123")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload struct {
		Albums []store.Album `json:"albums"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Albums) != 1 || payload.Albums[0].ID != 1 {
		t.Fatalf("unexpected albums payload: %#v", payload.Albums)
	}
	if albumStub.lastToken != "token-123" {
		t.Fatalf("expected token 'token-123', got %q", albumStub.lastToken)
	}
}

func TestHandleAlbumsGetUnauthorized(t *testing.T) {
	albumStub := &stubAlbumService{
		albumsErr: store.ErrUnauthorized,
	}

	server := newTestServer(t, albumStub, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/me/albums", nil)
	req.Header.Set("Authorization", "Bearer bad")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumsPostSuccess(t *testing.T) {
	albumStub := &stubAlbumService{}
	server := newTestServer(t, albumStub, nil, nil)

	body := albumRequest{
		Artist:      "Artist",
		Title:       "Title",
		ReleaseYear: 2024,
		Tracks:      []string{"Song"},
		Genres:      []string{"Indie"},
		Rating:      5,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/me/albums", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if albumStub.lastToken != "token" {
		t.Fatalf("expected token 'token', got %q", albumStub.lastToken)
	}
	if albumStub.createdAlbum.Artist != "Artist" || albumStub.createdAlbum.Rating != 5 {
		t.Fatalf("unexpected created album: %#v", albumStub.createdAlbum)
	}
}

func TestHandleAlbumsPostValidationError(t *testing.T) {
	albumStub := &stubAlbumService{
		createErr: store.ErrInvalidAlbum,
	}
	server := newTestServer(t, albumStub, nil, nil)

	body := albumRequest{
		Artist:      "Artist",
		Title:       "Title",
		ReleaseYear: 2024,
		Tracks:      []string{"Song"},
		Genres:      []string{"Indie"},
		Rating:      0,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/me/albums", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAlbumsPostMissingToken(t *testing.T) {
	server := newTestServer(t, &stubAlbumService{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/me/albums", bytes.NewReader([]byte(`{}`)))

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumsPostUnexpectedError(t *testing.T) {
	albumStub := &stubAlbumService{
		createErr: errors.New("boom"),
	}
	server := newTestServer(t, albumStub, nil, nil)

	body := albumRequest{
		Artist:      "Artist",
		Title:       "Title",
		ReleaseYear: 2024,
		Tracks:      []string{"Song"},
		Genres:      []string{"Indie"},
		Rating:      5,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/me/albums", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestHandleAlbumPreferencePut(t *testing.T) {
	ratingsStub := &stubRatingsService{}
	server := newTestServer(t, &stubAlbumService{}, ratingsStub, nil)

	body := albumPreferenceRequest{
		Rating:    ptr(4),
		Favorited: true,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/me/albums/10/preference", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if ratingsStub.lastToken != "token" || ratingsStub.lastAlbumID != 10 {
		t.Fatalf("unexpected store call: token=%q albumID=%d", ratingsStub.lastToken, ratingsStub.lastAlbumID)
	}
	if ratingsStub.lastRating == nil || *ratingsStub.lastRating != 4 || !ratingsStub.lastFavorited {
		t.Fatalf("unexpected preference data: rating=%v favorited=%v", ratingsStub.lastRating, ratingsStub.lastFavorited)
	}
}

func TestHandleAlbumPreferenceDelete(t *testing.T) {
	ratingsStub := &stubRatingsService{}
	server := newTestServer(t, &stubAlbumService{}, ratingsStub, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/me/albums/10/preference", nil)
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if ratingsStub.lastRating != nil || ratingsStub.lastFavorited {
		t.Fatalf("expected rating nil and favorited false, got rating=%v favorited=%v", ratingsStub.lastRating, ratingsStub.lastFavorited)
	}
}

func TestHandleAlbumPreferenceErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid", store.ErrInvalidAlbum, http.StatusBadRequest},
		{"notfound", store.ErrAlbumNotFound, http.StatusNotFound},
		{"unauthorized", store.ErrUnauthorized, http.StatusUnauthorized},
		{"other", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ratingsStub := &stubRatingsService{upsertErr: tc.err}
			server := newTestServer(t, &stubAlbumService{}, ratingsStub, nil)

			body := albumPreferenceRequest{}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPut, "/api/me/albums/5/preference", bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer token")

			rr := httptest.NewRecorder()
			server.Routes().ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}

func TestHandleAlbumPreferenceMissingToken(t *testing.T) {
	server := newTestServer(t, &stubAlbumService{}, &stubRatingsService{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/me/albums/10/preference", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumPreferencesList(t *testing.T) {
	rating := 5
	ratingsStub := &stubRatingsService{
		preferencesResponse: []store.AlbumPreference{
			{
				Album: store.Album{
					ID:          1,
					Artist:      "Artist",
					Title:       "Title",
					ReleaseYear: 2000,
					Rating:      4,
				},
				Rating:    &rating,
				Favorited: true,
			},
		},
	}
	server := newTestServer(t, &stubAlbumService{}, ratingsStub, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/me/albums/preferences", nil)
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload struct {
		Preferences []store.AlbumPreference `json:"preferences"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Preferences) != 1 || !payload.Preferences[0].Favorited {
		t.Fatalf("unexpected preferences: %#v", payload.Preferences)
	}
}

func TestHandleAlbumPreferencesUnauthorized(t *testing.T) {
	server := newTestServer(t, &stubAlbumService{}, &stubRatingsService{preferencesErr: store.ErrUnauthorized}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/me/albums/preferences", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestHandleAlbumsListSuccess(t *testing.T) {
	albumStub := &stubAlbumService{
		listAlbumsResponse: []store.Album{
			{ID: 10, Artist: "Boards of Canada", Title: "Geogaddi", ReleaseYear: 2002, Rating: 5},
		},
	}
	server := newTestServer(t, albumStub, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/albums?artist=boards&rating=5", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var payload struct {
		Albums []store.Album `json:"albums"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Albums) != 1 || payload.Albums[0].ID != 10 {
		t.Fatalf("unexpected albums: %#v", payload.Albums)
	}
}

func TestHandleAlbumsListBadRating(t *testing.T) {
	server := newTestServer(t, &stubAlbumService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/albums?rating=bad", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAlbumSuccess(t *testing.T) {
	albumStub := &stubAlbumService{
		singleAlbum: store.Album{ID: 5, Artist: "Artist"},
	}
	server := newTestServer(t, albumStub, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/album?id=5", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var album store.Album
	if err := json.NewDecoder(rr.Body).Decode(&album); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if album.ID != 5 {
		t.Fatalf("expected id 5, got %d", album.ID)
	}
}

func TestHandleAlbumNotFound(t *testing.T) {
	albumStub := &stubAlbumService{
		singleErr: store.ErrAlbumNotFound,
	}
	server := newTestServer(t, albumStub, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/album?id=999", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleFavoriteTrackPutCreated(t *testing.T) {
	trackID := int64(42)
	favoritedAt := time.Date(2024, time.April, 1, 10, 30, 0, 0, time.UTC)
	favoritesStub := &stubFavoritesService{
		favoriteTrackCreated: true,
		favoriteTrackResponse: &models.Favorite{
			ID:        7,
			UserID:    21,
			SongID:    &trackID,
			CreatedAt: favoritedAt,
		},
	}
	server := newTestServer(t, nil, nil, favoritesStub)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/favorites/tracks/42", nil)
	req.Header.Set("Authorization", "Bearer token-x")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if favoritesStub.lastFavoriteToken != "token-x" {
		t.Fatalf("expected token 'token-x', got %q", favoritesStub.lastFavoriteToken)
	}
	if favoritesStub.lastFavoriteTrackID != 42 {
		t.Fatalf("expected track ID 42, got %d", favoritesStub.lastFavoriteTrackID)
	}
	if got := rr.Header().Get("Location"); got != "/api/v1/me/favorites/tracks/42" {
		t.Fatalf("expected Location header, got %q", got)
	}

	var payload struct {
		Track struct {
			TrackID     int64     `json:"track_id"`
			FavoritedAt time.Time `json:"favorited_at"`
		} `json:"track"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Track.TrackID != 42 {
		t.Fatalf("expected track ID 42, got %d", payload.Track.TrackID)
	}
	if !payload.Track.FavoritedAt.Equal(favoritedAt) {
		t.Fatalf("expected favorited time %v, got %v", favoritedAt, payload.Track.FavoritedAt)
	}
}

func TestHandleFavoriteTrackPutCreatedLegacyPath(t *testing.T) {
	trackID := int64(17)
	favoritedAt := time.Date(2024, time.May, 15, 9, 0, 0, 0, time.UTC)
	favoritesStub := &stubFavoritesService{
		favoriteTrackCreated: true,
		favoriteTrackResponse: &models.Favorite{
			ID:        9,
			UserID:    2,
			SongID:    &trackID,
			CreatedAt: favoritedAt,
		},
	}
	server := newTestServer(t, nil, nil, favoritesStub)
	req := httptest.NewRequest(http.MethodPut, "/api/me/favorites/tracks/17", nil)
	req.Header.Set("Authorization", "Bearer legacy-token")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if favoritesStub.lastFavoriteToken != "legacy-token" {
		t.Fatalf("expected token 'legacy-token', got %q", favoritesStub.lastFavoriteToken)
	}
	if favoritesStub.lastFavoriteTrackID != 17 {
		t.Fatalf("expected track ID 17, got %d", favoritesStub.lastFavoriteTrackID)
	}
}

func TestHandleFavoriteTrackPutIdempotent(t *testing.T) {
	favoritesStub := &stubFavoritesService{
		favoriteTrackCreated: false,
	}
	server := newTestServer(t, nil, nil, favoritesStub)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/favorites/tracks/5", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if favoritesStub.lastFavoriteTrackID != 5 {
		t.Fatalf("expected track ID 5, got %d", favoritesStub.lastFavoriteTrackID)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body for 204, got %q", rr.Body.String())
	}
}

func TestHandleFavoriteTrackDeleteNotFound(t *testing.T) {
	favoritesStub := &stubFavoritesService{
		unfavoriteErr: store.ErrFavoriteNotFound,
	}
	server := newTestServer(t, nil, nil, favoritesStub)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/me/favorites/tracks/8", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
	var payload errorResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error == "" {
		t.Fatalf("expected error message in response")
	}
	if favoritesStub.lastUnfavoriteTrackID != 8 {
		t.Fatalf("expected track ID 8, got %d", favoritesStub.lastUnfavoriteTrackID)
	}
}

func TestHandleFavoriteTracksGetSuccess(t *testing.T) {
	trackID := int64(11)
	now := time.Date(2024, time.May, 2, 12, 0, 0, 0, time.UTC)
	favoritesStub := &stubFavoritesService{
		listResponse: []*models.Favorite{
			{ID: 1, UserID: 3, SongID: &trackID, CreatedAt: now},
		},
	}
	server := newTestServer(t, nil, nil, favoritesStub)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/favorites/tracks", nil)
	req.Header.Set("Authorization", "Bearer abc")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if favoritesStub.lastListToken != "abc" {
		t.Fatalf("expected token 'abc', got %q", favoritesStub.lastListToken)
	}
	var payload struct {
		Tracks []struct {
			TrackID     int64     `json:"track_id"`
			FavoritedAt time.Time `json:"favorited_at"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Tracks) != 1 || payload.Tracks[0].TrackID != 11 {
		t.Fatalf("unexpected tracks payload: %+v", payload.Tracks)
	}
}

func TestHandleFavoriteTrackPutInvalidID(t *testing.T) {
	server := newTestServer(t, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/favorites/tracks/not-a-number", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	var payload errorResponse
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error == "" {
		t.Fatalf("expected error message, got empty string")
	}
}
