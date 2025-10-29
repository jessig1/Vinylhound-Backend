package models

import "time"

// Favorite represents a user's favorited song or album (heart icon).
// Either SongID or AlbumID must be set, but not both.
type Favorite struct {
	ID        int64      `json:"id" db:"id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	SongID    *int64     `json:"song_id,omitempty" db:"song_id"`
	AlbumID   *int64     `json:"album_id,omitempty" db:"album_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// FavoriteRequest represents a request to favorite or unfavorite an item.
type FavoriteRequest struct {
	SongID  *int64 `json:"song_id,omitempty"`
	AlbumID *int64 `json:"album_id,omitempty"`
}
