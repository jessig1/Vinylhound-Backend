package models

import "time"

// Concert represents a music concert/performance
type Concert struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	VenueID     int64     `json:"venue_id"`
	ArtistName  string    `json:"artist_name"`           // Primary artist/headliner
	Name        string    `json:"name"`                  // Concert name/tour name
	Date        time.Time `json:"date"`                  // Concert date and time
	TicketPrice *float64  `json:"ticket_price,omitempty"` // Optional
	Notes       string    `json:"notes,omitempty"`       // User notes
	Attended    bool      `json:"attended"`              // Did user attend?
	Rating      *int      `json:"rating,omitempty"`      // User rating (1-5)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Populated via JOIN queries (not stored in concerts table)
	Venue *Venue `json:"venue,omitempty"` // Embedded venue details
}

// ConcertFilter for searching concerts
type ConcertFilter struct {
	ArtistName string
	VenueID    *int64
	City       string
	State      string
	FromDate   *time.Time // Concerts after this date
	ToDate     *time.Time // Concerts before this date
	Attended   *bool      // Filter by attendance status
}

// ConcertWithDetails includes full venue information
type ConcertWithDetails struct {
	Concert
	VenueName    string `json:"venue_name"`
	VenueAddress string `json:"venue_address"`
	VenueCity    string `json:"venue_city"`
	VenueState   string `json:"venue_state"`
}
