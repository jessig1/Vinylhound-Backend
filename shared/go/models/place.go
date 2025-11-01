package models

import "time"

// PlaceType distinguishes between venues and retailers
type PlaceType string

const (
	PlaceTypeVenue    PlaceType = "venue"
	PlaceTypeRetailer PlaceType = "retailer"
)

// Venue represents a music venue
type Venue struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Capacity    *int      `json:"capacity,omitempty"`    // Optional
	Description string    `json:"description,omitempty"` // Optional
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Retailer represents a record store or music retailer
type Retailer struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Specialty   string    `json:"specialty,omitempty"`   // e.g., "Vinyl", "CDs", "Cassettes"
	Website     string    `json:"website,omitempty"`     // Optional
	Description string    `json:"description,omitempty"` // Optional
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaceFilter for searching venues and retailers
type PlaceFilter struct {
	City      string
	State     string
	PlaceType *PlaceType // Optional filter by type
}
