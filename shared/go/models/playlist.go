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
	ID        int64          `json:"id" db:"id"`
	Title     string         `json:"title" db:"title"`
	Owner     string         `json:"owner" db:"owner"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
	SongCount int            `json:"song_count" db:"song_count"`
	Songs     []PlaylistSong `json:"songs"`
}
