package models

// UserPreference represents a user's album preference
type UserPreference struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"userId"`
	AlbumID   string `json:"albumId"`
	Rating    int    `json:"rating"`
	Favorited bool   `json:"favorited"`
}
