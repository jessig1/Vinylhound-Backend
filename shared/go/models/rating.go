package models

import (
	"time"
)

// Rating represents a user's rating of an album
type Rating struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	AlbumID   int64     `json:"album_id" db:"album_id"`
	Rating    int       `json:"rating" db:"rating"` // 1-5 stars
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Review represents a user's written review of an album
type Review struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	AlbumID   int64     `json:"album_id" db:"album_id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// GenrePreference represents user preferences for music genres
type GenrePreference struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Genre     string    `json:"genre" db:"genre"`
	Weight    float64   `json:"weight" db:"weight"` // 0.0 to 1.0
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RatingFilter represents filters for rating queries
type RatingFilter struct {
	UserID    int64 `json:"user_id"`
	AlbumID   int64 `json:"album_id"`
	MinRating int   `json:"min_rating"`
	MaxRating int   `json:"max_rating"`
	Limit     int   `json:"limit"`
	Offset    int   `json:"offset"`
}
