package models

import "time"

// CollectionType distinguishes between different types of album collections
type CollectionType string

const (
	CollectionTypeWishlist CollectionType = "wishlist"
	CollectionTypeOwned    CollectionType = "owned"
)

// AlbumCondition represents the physical condition of an owned album
type AlbumCondition string

const (
	ConditionMint     AlbumCondition = "mint"
	ConditionNearMint AlbumCondition = "near_mint"
	ConditionVeryGood AlbumCondition = "very_good"
	ConditionGood     AlbumCondition = "good"
	ConditionFair     AlbumCondition = "fair"
	ConditionPoor     AlbumCondition = "poor"
)

// AlbumCollection represents an album in a user's collection (wishlist or owned)
type AlbumCollection struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"user_id"`
	AlbumID        int64           `json:"album_id"`
	CollectionType CollectionType  `json:"collection_type"`
	Notes          string          `json:"notes,omitempty"`
	DateAdded      time.Time       `json:"date_added"`
	DateAcquired   *time.Time      `json:"date_acquired,omitempty"`   // When they got it (for owned albums)
	PurchasePrice  *float64        `json:"purchase_price,omitempty"`  // Optional purchase price
	Condition      *AlbumCondition `json:"condition,omitempty"`       // Physical condition (for owned albums)
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	// Populated via JOIN queries (not stored in album_collections table)
	Album *Album `json:"album,omitempty"` // Embedded album details
}

// AlbumCollectionWithDetails includes full album information
type AlbumCollectionWithDetails struct {
	AlbumCollection
	AlbumTitle      string `json:"album_title"`
	AlbumArtist     string `json:"album_artist"`
	AlbumReleaseYear int   `json:"album_release_year"`
	AlbumGenre      string `json:"album_genre"`
	AlbumCoverURL   string `json:"album_cover_url"`
}

// CollectionFilter for searching collections
type CollectionFilter struct {
	CollectionType *CollectionType
	Artist         string
	Genre          string
	YearFrom       *int
	YearTo         *int
	Condition      *AlbumCondition
	SearchTerm     string // Search in album title, artist, notes
	Limit          int
	Offset         int
}

// CollectionStats provides statistics about a user's collection
type CollectionStats struct {
	TotalWishlist int     `json:"total_wishlist"`
	TotalOwned    int     `json:"total_owned"`
	TotalValue    float64 `json:"total_value"` // Sum of purchase prices
	ByGenre       map[string]int `json:"by_genre,omitempty"`
	ByCondition   map[string]int `json:"by_condition,omitempty"`
}
