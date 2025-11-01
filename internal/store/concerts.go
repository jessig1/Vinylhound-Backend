package store

import (
	"context"
	"database/sql"
	"errors"

	"vinylhound/shared/go/models"
)

var (
	ErrConcertNotFound = errors.New("concert not found")
	ErrVenueRequired   = errors.New("venue_id is required")
)

// CreateConcert adds a new concert for a user
func (s *Store) CreateConcert(ctx context.Context, token string, concert *models.Concert) (*models.Concert, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO concerts (user_id, venue_id, artist_name, name, date,
		                     ticket_price, notes, attended, rating)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		userID, concert.VenueID, concert.ArtistName, concert.Name, concert.Date,
		concert.TicketPrice, concert.Notes, concert.Attended, concert.Rating,
	).Scan(&concert.ID, &concert.CreatedAt, &concert.UpdatedAt)

	if err != nil {
		return nil, err
	}

	concert.UserID = userID
	return concert, nil
}

// ListConcertsByUser returns all concerts for a user, optionally with venue details
func (s *Store) ListConcertsByUser(ctx context.Context, token string, includeVenue bool) ([]*models.ConcertWithDetails, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.id, c.user_id, c.venue_id, c.artist_name, c.name, c.date,
			c.ticket_price, c.notes, c.attended, c.rating,
			c.created_at, c.updated_at,
			v.name as venue_name, v.address as venue_address,
			v.city as venue_city, v.state as venue_state
		FROM concerts c
		INNER JOIN venues v ON c.venue_id = v.id
		WHERE c.user_id = $1
		ORDER BY c.date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []*models.ConcertWithDetails
	for rows.Next() {
		var c models.ConcertWithDetails
		err := rows.Scan(
			&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
			&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
			&c.CreatedAt, &c.UpdatedAt,
			&c.VenueName, &c.VenueAddress, &c.VenueCity, &c.VenueState,
		)
		if err != nil {
			return nil, err
		}
		concerts = append(concerts, &c)
	}

	return concerts, rows.Err()
}

// GetConcert retrieves a single concert by ID with venue details
func (s *Store) GetConcert(ctx context.Context, id int64) (*models.ConcertWithDetails, error) {
	query := `
		SELECT
			c.id, c.user_id, c.venue_id, c.artist_name, c.name, c.date,
			c.ticket_price, c.notes, c.attended, c.rating,
			c.created_at, c.updated_at,
			v.name as venue_name, v.address as venue_address,
			v.city as venue_city, v.state as venue_state
		FROM concerts c
		INNER JOIN venues v ON c.venue_id = v.id
		WHERE c.id = $1
	`

	var c models.ConcertWithDetails
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
		&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
		&c.CreatedAt, &c.UpdatedAt,
		&c.VenueName, &c.VenueAddress, &c.VenueCity, &c.VenueState,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConcertNotFound
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// UpdateConcert updates an existing concert
func (s *Store) UpdateConcert(ctx context.Context, token string, id int64, concert *models.Concert) (*models.Concert, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE concerts
		SET venue_id = $1, artist_name = $2, name = $3, date = $4,
		    ticket_price = $5, notes = $6, attended = $7, rating = $8,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $9 AND user_id = $10
		RETURNING id, user_id, venue_id, artist_name, name, date,
		          ticket_price, notes, attended, rating, created_at, updated_at
	`

	var c models.Concert
	err = s.db.QueryRowContext(ctx, query,
		concert.VenueID, concert.ArtistName, concert.Name, concert.Date,
		concert.TicketPrice, concert.Notes, concert.Attended, concert.Rating,
		id, userID,
	).Scan(
		&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
		&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
		&c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConcertNotFound
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// DeleteConcert removes a concert
func (s *Store) DeleteConcert(ctx context.Context, token string, id int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	query := `DELETE FROM concerts WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrConcertNotFound
	}

	return nil
}

// ListUpcomingConcerts returns future concerts for a user
func (s *Store) ListUpcomingConcerts(ctx context.Context, token string) ([]*models.ConcertWithDetails, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.id, c.user_id, c.venue_id, c.artist_name, c.name, c.date,
			c.ticket_price, c.notes, c.attended, c.rating,
			c.created_at, c.updated_at,
			v.name as venue_name, v.address as venue_address,
			v.city as venue_city, v.state as venue_state
		FROM concerts c
		INNER JOIN venues v ON c.venue_id = v.id
		WHERE c.user_id = $1 AND c.date >= CURRENT_TIMESTAMP
		ORDER BY c.date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []*models.ConcertWithDetails
	for rows.Next() {
		var c models.ConcertWithDetails
		err := rows.Scan(
			&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
			&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
			&c.CreatedAt, &c.UpdatedAt,
			&c.VenueName, &c.VenueAddress, &c.VenueCity, &c.VenueState,
		)
		if err != nil {
			return nil, err
		}
		concerts = append(concerts, &c)
	}

	return concerts, rows.Err()
}

// ListConcertsByVenue returns all concerts at a specific venue
func (s *Store) ListConcertsByVenue(ctx context.Context, venueID int64) ([]*models.ConcertWithDetails, error) {
	query := `
		SELECT
			c.id, c.user_id, c.venue_id, c.artist_name, c.name, c.date,
			c.ticket_price, c.notes, c.attended, c.rating,
			c.created_at, c.updated_at,
			v.name as venue_name, v.address as venue_address,
			v.city as venue_city, v.state as venue_state
		FROM concerts c
		INNER JOIN venues v ON c.venue_id = v.id
		WHERE c.venue_id = $1
		ORDER BY c.date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []*models.ConcertWithDetails
	for rows.Next() {
		var c models.ConcertWithDetails
		err := rows.Scan(
			&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
			&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
			&c.CreatedAt, &c.UpdatedAt,
			&c.VenueName, &c.VenueAddress, &c.VenueCity, &c.VenueState,
		)
		if err != nil {
			return nil, err
		}
		concerts = append(concerts, &c)
	}

	return concerts, rows.Err()
}

// ListConcertsByArtist returns all concerts for a specific artist
func (s *Store) ListConcertsByArtist(ctx context.Context, token string, artistName string) ([]*models.ConcertWithDetails, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.id, c.user_id, c.venue_id, c.artist_name, c.name, c.date,
			c.ticket_price, c.notes, c.attended, c.rating,
			c.created_at, c.updated_at,
			v.name as venue_name, v.address as venue_address,
			v.city as venue_city, v.state as venue_state
		FROM concerts c
		INNER JOIN venues v ON c.venue_id = v.id
		WHERE c.user_id = $1 AND LOWER(c.artist_name) = LOWER($2)
		ORDER BY c.date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, artistName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []*models.ConcertWithDetails
	for rows.Next() {
		var c models.ConcertWithDetails
		err := rows.Scan(
			&c.ID, &c.UserID, &c.VenueID, &c.ArtistName, &c.Name, &c.Date,
			&c.TicketPrice, &c.Notes, &c.Attended, &c.Rating,
			&c.CreatedAt, &c.UpdatedAt,
			&c.VenueName, &c.VenueAddress, &c.VenueCity, &c.VenueState,
		)
		if err != nil {
			return nil, err
		}
		concerts = append(concerts, &c)
	}

	return concerts, rows.Err()
}

// MarkConcertAttended marks a concert as attended and optionally adds a rating
func (s *Store) MarkConcertAttended(ctx context.Context, token string, concertID int64, rating *int) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	query := `
		UPDATE concerts
		SET attended = TRUE, rating = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND user_id = $3
	`

	result, err := s.db.ExecContext(ctx, query, rating, concertID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrConcertNotFound
	}

	return nil
}
