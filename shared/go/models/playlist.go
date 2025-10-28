package models

import "time"

// PlaylistSong represents a song inside a playlist with relevant metadata.
type PlaylistSong struct {
	ID            int64  `json:"id" db:"id"`
	Title         string `json:"title" db:"title"`
	Artist        string `json:"artist" db:"artist"`
	Album         string `json:"album" db:"album"`
	LengthSeconds int    `json:"length_seconds" db:"length_seconds"`
	Genre         string `json:"genre" db:"genre"`
}

// Playlist captures a user-curated list of songs.
type Playlist struct {
	ID          int64          `json:"id" db:"id"`
	Title       string         `json:"title" db:"title"`
	Description string         `json:"description,omitempty" db:"description"`
	Owner       string         `json:"owner" db:"owner"`
	UserID      int64          `json:"user_id,omitempty" db:"user_id"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	SongCount   int            `json:"song_count" db:"song_count"`
	Tags        []string       `json:"tags" db:"tags"`
	IsPublic    bool           `json:"is_public" db:"is_public"`
	Songs       []PlaylistSong `json:"songs"`
}
