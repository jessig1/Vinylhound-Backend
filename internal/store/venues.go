package store

import (
	"context"
	"database/sql"
	"errors"

	"vinylhound/shared/go/models"
)

var (
	ErrVenueNotFound    = errors.New("venue not found")
	ErrRetailerNotFound = errors.New("retailer not found")
)

// CreateVenue adds a new venue for a user
func (s *Store) CreateVenue(ctx context.Context, token string, venue *models.Venue) (*models.Venue, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO venues (user_id, name, address, city, state, capacity, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		userID, venue.Name, venue.Address, venue.City, venue.State,
		venue.Capacity, venue.Description,
	).Scan(&venue.ID, &venue.CreatedAt, &venue.UpdatedAt)

	if err != nil {
		return nil, err
	}

	venue.UserID = userID
	return venue, nil
}

// ListVenuesByUser returns all venues for a user
func (s *Store) ListVenuesByUser(ctx context.Context, token string) ([]*models.Venue, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, name, address, city, state, capacity, description,
		       created_at, updated_at
		FROM venues
		WHERE user_id = $1
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []*models.Venue
	for rows.Next() {
		var v models.Venue
		err := rows.Scan(&v.ID, &v.UserID, &v.Name, &v.Address, &v.City,
			&v.State, &v.Capacity, &v.Description, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		venues = append(venues, &v)
	}

	return venues, rows.Err()
}

// GetVenue retrieves a single venue by ID
func (s *Store) GetVenue(ctx context.Context, id int64) (*models.Venue, error) {
	query := `
		SELECT id, user_id, name, address, city, state, capacity, description,
		       created_at, updated_at
		FROM venues
		WHERE id = $1
	`

	var v models.Venue
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&v.ID, &v.UserID, &v.Name, &v.Address, &v.City, &v.State,
		&v.Capacity, &v.Description, &v.CreatedAt, &v.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrVenueNotFound
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// UpdateVenue updates an existing venue
func (s *Store) UpdateVenue(ctx context.Context, token string, id int64, venue *models.Venue) (*models.Venue, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE venues
		SET name = $1, address = $2, city = $3, state = $4,
		    capacity = $5, description = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7 AND user_id = $8
		RETURNING id, user_id, name, address, city, state, capacity, description,
		          created_at, updated_at
	`

	var v models.Venue
	err = s.db.QueryRowContext(ctx, query,
		venue.Name, venue.Address, venue.City, venue.State,
		venue.Capacity, venue.Description, id, userID,
	).Scan(&v.ID, &v.UserID, &v.Name, &v.Address, &v.City, &v.State,
		&v.Capacity, &v.Description, &v.CreatedAt, &v.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrVenueNotFound
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// DeleteVenue removes a venue
func (s *Store) DeleteVenue(ctx context.Context, token string, id int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	query := `DELETE FROM venues WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrVenueNotFound
	}

	return nil
}

// CreateRetailer adds a new retailer for a user
func (s *Store) CreateRetailer(ctx context.Context, token string, retailer *models.Retailer) (*models.Retailer, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO retailers (user_id, name, address, city, state, specialty, website, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		userID, retailer.Name, retailer.Address, retailer.City, retailer.State,
		retailer.Specialty, retailer.Website, retailer.Description,
	).Scan(&retailer.ID, &retailer.CreatedAt, &retailer.UpdatedAt)

	if err != nil {
		return nil, err
	}

	retailer.UserID = userID
	return retailer, nil
}

// ListRetailersByUser returns all retailers for a user
func (s *Store) ListRetailersByUser(ctx context.Context, token string) ([]*models.Retailer, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, name, address, city, state, specialty, website, description,
		       created_at, updated_at
		FROM retailers
		WHERE user_id = $1
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retailers []*models.Retailer
	for rows.Next() {
		var r models.Retailer
		err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Address, &r.City,
			&r.State, &r.Specialty, &r.Website, &r.Description, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		retailers = append(retailers, &r)
	}

	return retailers, rows.Err()
}

// GetRetailer retrieves a single retailer by ID
func (s *Store) GetRetailer(ctx context.Context, id int64) (*models.Retailer, error) {
	query := `
		SELECT id, user_id, name, address, city, state, specialty, website, description,
		       created_at, updated_at
		FROM retailers
		WHERE id = $1
	`

	var r models.Retailer
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.UserID, &r.Name, &r.Address, &r.City, &r.State,
		&r.Specialty, &r.Website, &r.Description, &r.CreatedAt, &r.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRetailerNotFound
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// UpdateRetailer updates an existing retailer
func (s *Store) UpdateRetailer(ctx context.Context, token string, id int64, retailer *models.Retailer) (*models.Retailer, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE retailers
		SET name = $1, address = $2, city = $3, state = $4,
		    specialty = $5, website = $6, description = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND user_id = $9
		RETURNING id, user_id, name, address, city, state, specialty, website, description,
		          created_at, updated_at
	`

	var r models.Retailer
	err = s.db.QueryRowContext(ctx, query,
		retailer.Name, retailer.Address, retailer.City, retailer.State,
		retailer.Specialty, retailer.Website, retailer.Description, id, userID,
	).Scan(&r.ID, &r.UserID, &r.Name, &r.Address, &r.City, &r.State,
		&r.Specialty, &r.Website, &r.Description, &r.CreatedAt, &r.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrRetailerNotFound
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// DeleteRetailer removes a retailer
func (s *Store) DeleteRetailer(ctx context.Context, token string, id int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	query := `DELETE FROM retailers WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRetailerNotFound
	}

	return nil
}
