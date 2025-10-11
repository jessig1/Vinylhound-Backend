package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vinylhound/internal/store"
)

type stubStore struct {
	albumsResponse []store.Album
	albumsErr      error

	createdAlbum store.Album
	createErr    error

	listAlbumsResponse []store.Album
	listAlbumsErr      error

	singleAlbum store.Album
	singleErr   error

	preferencesResponse []store.AlbumPreference
	preferencesErr      error

	upsertErr     error
	lastAlbumID   int64
	lastRating    *int
	lastFavorited bool

	lastToken string
}

func (s *stubStore) CreateUser(username, password string, content []string) error {
	return nil
}

func (s *stubStore) Authenticate(username, password string) (string, error) {
	return "", nil
}

func (s *stubStore) ContentByToken(token string) ([]string, error) {
	return nil, nil
}

func (s *stubStore) UpdateContentByToken(token string, content []string) error {
	return nil
}

func (s *stubStore) CreateAlbum(token string, album store.Album) (store.Album, error) {
	s.lastToken = token
	s.createdAlbum = album
	if s.createErr != nil {
		return store.Album{}, s.createErr
	}
	return s.createdAlbum, nil
}

func (s *stubStore) AlbumsByToken(token string) ([]store.Album, error) {
	s.lastToken = token
	if s.albumsErr != nil {
		return nil, s.albumsErr
	}
	return s.albumsResponse, nil
}

func (s *stubStore) ListAlbums(filter store.AlbumFilter) ([]store.Album, error) {
	if s.listAlbumsErr != nil {
		return nil, s.listAlbumsErr
	}
	return s.listAlbumsResponse, nil
}

func (s *stubStore) AlbumByID(id int64) (store.Album, error) {
	if s.singleErr != nil {
		return store.Album{}, s.singleErr
	}
	return s.singleAlbum, nil
}

func (s *stubStore) UpsertAlbumPreference(token string, albumID int64, rating *int, favorited bool) error {
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

func (s *stubStore) AlbumPreferencesByToken(token string) ([]store.AlbumPreference, error) {
	s.lastToken = token
	if s.preferencesErr != nil {
		return nil, s.preferencesErr
	}
	return s.preferencesResponse, nil
}

func TestHandleAlbumsGetSuccess(t *testing.T) {
	stub := &stubStore{
		albumsResponse: []store.Album{
			{ID: 1, Artist: "Artist", Title: "Title", ReleaseYear: 2000, Rating: 4},
		},
	}

	server := New(stub)
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
	if stub.lastToken != "token-123" {
		t.Fatalf("expected token 'token-123', got %q", stub.lastToken)
	}
}

func TestHandleAlbumsGetUnauthorized(t *testing.T) {
	stub := &stubStore{
		albumsErr: store.ErrUnauthorized,
	}

	server := New(stub)
	req := httptest.NewRequest(http.MethodGet, "/api/me/albums", nil)
	req.Header.Set("Authorization", "Bearer bad")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumsPostSuccess(t *testing.T) {
	stub := &stubStore{}
	server := New(stub)

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
	if stub.lastToken != "token" {
		t.Fatalf("expected token 'token', got %q", stub.lastToken)
	}
	if stub.createdAlbum.Artist != "Artist" || stub.createdAlbum.Rating != 5 {
		t.Fatalf("unexpected created album: %#v", stub.createdAlbum)
	}
}

func TestHandleAlbumsPostValidationError(t *testing.T) {
	stub := &stubStore{
		createErr: store.ErrInvalidAlbum,
	}
	server := New(stub)

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
	server := New(&stubStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/me/albums", bytes.NewReader([]byte(`{}`)))

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumsPostUnexpectedError(t *testing.T) {
	stub := &stubStore{
		createErr: errors.New("boom"),
	}
	server := New(stub)

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
	stub := &stubStore{}
	server := New(stub)

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
	if stub.lastToken != "token" || stub.lastAlbumID != 10 {
		t.Fatalf("unexpected store call: token=%q albumID=%d", stub.lastToken, stub.lastAlbumID)
	}
	if stub.lastRating == nil || *stub.lastRating != 4 || !stub.lastFavorited {
		t.Fatalf("unexpected preference data: rating=%v favorited=%v", stub.lastRating, stub.lastFavorited)
	}
}

func TestHandleAlbumPreferenceDelete(t *testing.T) {
	stub := &stubStore{}
	server := New(stub)

	req := httptest.NewRequest(http.MethodDelete, "/api/me/albums/10/preference", nil)
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if stub.lastRating != nil || stub.lastFavorited {
		t.Fatalf("expected rating nil and favorited false, got rating=%v favorited=%v", stub.lastRating, stub.lastFavorited)
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
			stub := &stubStore{upsertErr: tc.err}
			server := New(stub)

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
	server := New(&stubStore{})
	req := httptest.NewRequest(http.MethodPut, "/api/me/albums/10/preference", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestHandleAlbumPreferencesList(t *testing.T) {
	rating := 5
	stub := &stubStore{
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
	server := New(stub)

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
	server := New(&stubStore{})
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
	stub := &stubStore{
		listAlbumsResponse: []store.Album{
			{ID: 10, Artist: "Boards of Canada", Title: "Geogaddi", ReleaseYear: 2002, Rating: 5},
		},
	}
	server := New(stub)

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
	server := New(&stubStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/albums?rating=bad", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleAlbumSuccess(t *testing.T) {
	stub := &stubStore{
		singleAlbum: store.Album{ID: 5, Artist: "Artist"},
	}
	server := New(stub)

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
	stub := &stubStore{
		singleErr: store.ErrAlbumNotFound,
	}
	server := New(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/album?id=999", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}
