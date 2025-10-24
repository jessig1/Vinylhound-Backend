package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Session represents a user session
type Session struct {
	Token     string    `json:"token" db:"token"`
	UserID    int64     `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// UserContent represents user-specific content
type UserContent struct {
	ID       int64  `json:"id" db:"id"`
	UserID   int64  `json:"user_id" db:"user_id"`
	Position int    `json:"position" db:"position"`
	Entry    string `json:"entry" db:"entry"`
}
