package store

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidateAlbum(t *testing.T) {
	tests := []struct {
		name    string
		album   Album
		wantErr bool
	}{
		{
			name: "valid album",
			album: Album{
				Artist:      "Aphex Twin",
				Title:       "Selected Ambient Works",
				ReleaseYear: 1992,
				Tracks:      []string{"Xtal"},
				Genres:      []string{"Ambient"},
				Rating:      5,
			},
		},
		{
			name: "missing artist",
			album: Album{
				Title:       "No Artist",
				ReleaseYear: 2020,
				Rating:      3,
			},
			wantErr: true,
		},
		{
			name: "invalid rating",
			album: Album{
				Artist:      "Artist",
				Title:       "Title",
				ReleaseYear: 2020,
				Rating:      6,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateAlbum(tc.album)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error but got %v", err)
			}
		})
	}
}

func TestCreateAlbumSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("token").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO albums (user_id, artist, title, release_year, tracks, genres, rating)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
		RETURNING id
	`)).
		WithArgs(int64(42), "Artist", "Title", 1999, `["Track 1"]`, `["Electronic"]`, 4).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))

	album := Album{
		Artist:      "  Artist ",
		Title:       " Title  ",
		ReleaseYear: 1999,
		Tracks:      []string{"Track 1"},
		Genres:      []string{"Electronic"},
		Rating:      4,
	}

	got, err := s.CreateAlbum("token", album)
	if err != nil {
		t.Fatalf("CreateAlbum error: %v", err)
	}

	if got.ID != 99 {
		t.Fatalf("expected album ID 99, got %d", got.ID)
	}
	if got.Artist != "Artist" || got.Title != "Title" {
		t.Fatalf("expected trimmed artist/title, got %q / %q", got.Artist, got.Title)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateAlbumUnauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("bad-token").
		WillReturnError(sql.ErrNoRows)

	_, err = s.CreateAlbum("bad-token", Album{
		Artist:      "Artist",
		Title:       "Title",
		ReleaseYear: 2000,
		Rating:      3,
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListAlbumsWithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	expectedQuery := regexp.QuoteMeta(`
		SELECT id, artist, title, release_year, tracks, genres, rating
		FROM albums WHERE artist ILIKE $1 AND rating = $2 ORDER BY release_year DESC, id ASC
	`)

	mock.ExpectQuery(expectedQuery).
		WithArgs("%Boards%", 5).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "artist", "title", "release_year", "tracks", "genres", "rating",
		}).AddRow(int64(1), "Boards of Canada", "Geogaddi", 2002, `["Music Is Math"]`, `["Electronic"]`, 5))

	albums, err := s.ListAlbums(AlbumFilter{
		Artist: "Boards",
		Rating: 5,
	})
	if err != nil {
		t.Fatalf("ListAlbums error: %v", err)
	}

	if len(albums) != 1 || albums[0].Title != "Geogaddi" {
		t.Fatalf("unexpected albums: %#v", albums)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAlbumByIDNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, artist, title, release_year, tracks, genres, rating
		FROM albums
		WHERE id = $1
	`)).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err = s.AlbumByID(999)
	if !errors.Is(err, ErrAlbumNotFound) {
		t.Fatalf("expected ErrAlbumNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertAlbumPreferenceInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("token").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id
		FROM albums
		WHERE id = $1
	`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO user_album_preferences (user_id, album_id, rating, favorited, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, album_id)
		DO UPDATE SET rating = EXCLUDED.rating, favorited = EXCLUDED.favorited, updated_at = NOW()
	`)).
		WithArgs(int64(42), int64(10), 5, true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rating := 5
	if err := s.UpsertAlbumPreference("token", 10, &rating, true); err != nil {
		t.Fatalf("UpsertAlbumPreference: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertAlbumPreferenceDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("token").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id
		FROM albums
		WHERE id = $1
	`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM user_album_preferences
		WHERE user_id = $1 AND album_id = $2
	`)).
		WithArgs(int64(42), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := s.UpsertAlbumPreference("token", 10, nil, false); err != nil {
		t.Fatalf("UpsertAlbumPreference delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertAlbumPreferenceInvalidRating(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	rating := 6
	if err := s.UpsertAlbumPreference("token", 10, &rating, true); !errors.Is(err, ErrInvalidAlbum) {
		t.Fatalf("expected ErrInvalidAlbum, got %v", err)
	}
}

func TestUpsertAlbumPreferenceAlbumNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("token").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id
		FROM albums
		WHERE id = $1
	`)).
		WithArgs(int64(10)).
		WillReturnError(sql.ErrNoRows)

	rating := 4
	if err := s.UpsertAlbumPreference("token", 10, &rating, false); !errors.Is(err, ErrAlbumNotFound) {
		t.Fatalf("expected ErrAlbumNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAlbumPreferencesByToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`)).
		WithArgs("token").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			a.id, a.artist, a.title, a.release_year, a.tracks, a.genres, a.rating,
			p.rating, p.favorited
		FROM user_album_preferences p
		JOIN albums a ON a.id = p.album_id
		WHERE p.user_id = $1
		ORDER BY p.updated_at DESC
	`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "artist", "title", "release_year", "tracks", "genres", "rating", "user_rating", "favorited",
		}).AddRow(
			int64(1),
			"Artist",
			"Title",
			2000,
			`["Track"]`,
			`["Genre"]`,
			4,
			int64(5),
			true,
		))

	prefs, err := s.AlbumPreferencesByToken("token")
	if err != nil {
		t.Fatalf("AlbumPreferencesByToken: %v", err)
	}

	if len(prefs) != 1 {
		t.Fatalf("expected 1 preference, got %d", len(prefs))
	}
	if prefs[0].Album.ID != 1 || prefs[0].Rating == nil || *prefs[0].Rating != 5 || !prefs[0].Favorited {
		t.Fatalf("unexpected preference: %#v", prefs[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
