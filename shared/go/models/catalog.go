package models

import (
	"time"
)

// Album represents a music album
type Album struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Artist      string    `json:"artist" db:"artist"`
	ReleaseYear int       `json:"release_year" db:"release_year"`
	Genre       string    `json:"genre" db:"genre"`
	CoverURL    string    `json:"cover_url" db:"cover_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Artist represents a musical artist
type Artist struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Biography string    `json:"biography" db:"biography"`
	ImageURL  string    `json:"image_url" db:"image_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Song represents a song
type Song struct {
	ID        int64     `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Artist    string    `json:"artist" db:"artist"`
	AlbumID   int64     `json:"album_id" db:"album_id"`
	Duration  int       `json:"duration" db:"duration"` // in seconds
	TrackNum  int       `json:"track_num" db:"track_num"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AlbumFilter represents filters for album queries
type AlbumFilter struct {
	Artist     string `json:"artist"`
	Genre      string `json:"genre"`
	YearFrom   int    `json:"year_from"`
	YearTo     int    `json:"year_to"`
	SearchTerm string `json:"search_term"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}
