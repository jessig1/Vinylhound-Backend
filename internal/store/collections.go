package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"vinylhound/shared/go/models"
)

var (
	ErrCollectionNotFound      = errors.New("collection item not found")
	ErrAlreadyInCollection     = errors.New("album already in this collection")
	ErrInvalidCollectionType   = errors.New("invalid collection type")
)

// AddToCollection adds an album to a user's collection (wishlist or owned)
func (s *Store) AddToCollection(ctx context.Context, token string, collection *models.AlbumCollection) (*models.AlbumCollection, error) {
	if collection == nil {
		return nil, errors.New("collection item is required")
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Validate collection type
	if collection.CollectionType != models.CollectionTypeWishlist && collection.CollectionType != models.CollectionTypeOwned {
		return nil, ErrInvalidCollectionType
	}

	// Verify album exists
	var albumExists bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM albums WHERE id = $1)`, collection.AlbumID).Scan(&albumExists)
	if err != nil {
		return nil, fmt.Errorf("check album existence: %w", err)
	}
	if !albumExists {
		return nil, errors.New("album not found")
	}

	now := time.Now().UTC()
	collection.UserID = userID
	collection.DateAdded = now
	collection.CreatedAt = now
	collection.UpdatedAt = now

	var notes, condition sql.NullString
	var dateAcquired sql.NullTime
	var purchasePrice sql.NullFloat64

	if collection.Notes != "" {
		notes = sql.NullString{String: collection.Notes, Valid: true}
	}
	if collection.DateAcquired != nil {
		dateAcquired = sql.NullTime{Time: *collection.DateAcquired, Valid: true}
	}
	if collection.PurchasePrice != nil {
		purchasePrice = sql.NullFloat64{Float64: *collection.PurchasePrice, Valid: true}
	}
	if collection.Condition != nil {
		condition = sql.NullString{String: string(*collection.Condition), Valid: true}
	}

	err = s.db.QueryRowContext(ctx, `
		INSERT INTO album_collections (user_id, album_id, collection_type, notes, date_added, date_acquired, purchase_price, condition, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id, date_added, created_at, updated_at`,
		userID, collection.AlbumID, collection.CollectionType, notes, now, dateAcquired, purchasePrice, condition, now,
	).Scan(&collection.ID, &collection.DateAdded, &collection.CreatedAt, &collection.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyInCollection
		}
		return nil, fmt.Errorf("insert collection item: %w", err)
	}

	return collection, nil
}

// ListCollection returns all items in a user's collection with optional filtering
func (s *Store) ListCollection(ctx context.Context, token string, filter models.CollectionFilter) ([]*models.AlbumCollectionWithDetails, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			ac.id, ac.user_id, ac.album_id, ac.collection_type,
			COALESCE(ac.notes, ''), ac.date_added, ac.date_acquired, ac.purchase_price, ac.condition,
			ac.created_at, ac.updated_at,
			a.title, a.artist, a.release_year, COALESCE(a.genre, ''), COALESCE(a.cover_url, '')
		FROM album_collections ac
		JOIN albums a ON ac.album_id = a.id
		WHERE ac.user_id = $1`

	args := []interface{}{userID}
	argPos := 2

	if filter.CollectionType != nil {
		query += fmt.Sprintf(" AND ac.collection_type = $%d", argPos)
		args = append(args, string(*filter.CollectionType))
		argPos++
	}

	if filter.Artist != "" {
		query += fmt.Sprintf(" AND LOWER(a.artist) LIKE $%d", argPos)
		args = append(args, "%"+strings.ToLower(filter.Artist)+"%")
		argPos++
	}

	if filter.Genre != "" {
		query += fmt.Sprintf(" AND LOWER(a.genre) LIKE $%d", argPos)
		args = append(args, "%"+strings.ToLower(filter.Genre)+"%")
		argPos++
	}

	if filter.YearFrom != nil {
		query += fmt.Sprintf(" AND a.release_year >= $%d", argPos)
		args = append(args, *filter.YearFrom)
		argPos++
	}

	if filter.YearTo != nil {
		query += fmt.Sprintf(" AND a.release_year <= $%d", argPos)
		args = append(args, *filter.YearTo)
		argPos++
	}

	if filter.Condition != nil {
		query += fmt.Sprintf(" AND ac.condition = $%d", argPos)
		args = append(args, string(*filter.Condition))
		argPos++
	}

	if filter.SearchTerm != "" {
		query += fmt.Sprintf(" AND (LOWER(a.title) LIKE $%d OR LOWER(a.artist) LIKE $%d OR LOWER(ac.notes) LIKE $%d)", argPos, argPos, argPos)
		args = append(args, "%"+strings.ToLower(filter.SearchTerm)+"%")
		argPos++
	}

	query += " ORDER BY ac.date_added DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list collection: %w", err)
	}
	defer rows.Close()

	var items []*models.AlbumCollectionWithDetails
	for rows.Next() {
		var item models.AlbumCollectionWithDetails
		var notes, condition, coverURL sql.NullString
		var dateAcquired sql.NullTime
		var purchasePrice sql.NullFloat64

		err := rows.Scan(
			&item.ID, &item.UserID, &item.AlbumID, &item.CollectionType,
			&notes, &item.DateAdded, &dateAcquired, &purchasePrice, &condition,
			&item.CreatedAt, &item.UpdatedAt,
			&item.AlbumTitle, &item.AlbumArtist, &item.AlbumReleaseYear, &item.AlbumGenre, &coverURL,
		)
		if err != nil {
			return nil, fmt.Errorf("scan collection item: %w", err)
		}

		item.Notes = notes.String
		item.AlbumCoverURL = coverURL.String
		if dateAcquired.Valid {
			item.DateAcquired = &dateAcquired.Time
		}
		if purchasePrice.Valid {
			item.PurchasePrice = &purchasePrice.Float64
		}
		if condition.Valid {
			cond := models.AlbumCondition(condition.String)
			item.Condition = &cond
		}

		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate collection: %w", err)
	}

	return items, nil
}

// GetCollectionItem returns a single collection item by ID
func (s *Store) GetCollectionItem(ctx context.Context, id int64) (*models.AlbumCollectionWithDetails, error) {
	var item models.AlbumCollectionWithDetails
	var notes, condition, coverURL sql.NullString
	var dateAcquired sql.NullTime
	var purchasePrice sql.NullFloat64

	err := s.db.QueryRowContext(ctx, `
		SELECT
			ac.id, ac.user_id, ac.album_id, ac.collection_type,
			COALESCE(ac.notes, ''), ac.date_added, ac.date_acquired, ac.purchase_price, ac.condition,
			ac.created_at, ac.updated_at,
			a.title, a.artist, a.release_year, COALESCE(a.genre, ''), COALESCE(a.cover_url, '')
		FROM album_collections ac
		JOIN albums a ON ac.album_id = a.id
		WHERE ac.id = $1`, id).Scan(
		&item.ID, &item.UserID, &item.AlbumID, &item.CollectionType,
		&notes, &item.DateAdded, &dateAcquired, &purchasePrice, &condition,
		&item.CreatedAt, &item.UpdatedAt,
		&item.AlbumTitle, &item.AlbumArtist, &item.AlbumReleaseYear, &item.AlbumGenre, &coverURL,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCollectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get collection item: %w", err)
	}

	item.Notes = notes.String
	item.AlbumCoverURL = coverURL.String
	if dateAcquired.Valid {
		item.DateAcquired = &dateAcquired.Time
	}
	if purchasePrice.Valid {
		item.PurchasePrice = &purchasePrice.Float64
	}
	if condition.Valid {
		cond := models.AlbumCondition(condition.String)
		item.Condition = &cond
	}

	return &item, nil
}

// UpdateCollectionItem updates a collection item
func (s *Store) UpdateCollectionItem(ctx context.Context, token string, id int64, collection *models.AlbumCollection) (*models.AlbumCollection, error) {
	if collection == nil {
		return nil, errors.New("collection item is required")
	}

	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	var ownerID int64
	err = s.db.QueryRowContext(ctx, `SELECT user_id FROM album_collections WHERE id = $1`, id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCollectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check ownership: %w", err)
	}
	if ownerID != userID {
		return nil, errors.New("not authorized to modify this collection item")
	}

	var notes, condition sql.NullString
	var dateAcquired sql.NullTime
	var purchasePrice sql.NullFloat64

	if collection.Notes != "" {
		notes = sql.NullString{String: collection.Notes, Valid: true}
	}
	if collection.DateAcquired != nil {
		dateAcquired = sql.NullTime{Time: *collection.DateAcquired, Valid: true}
	}
	if collection.PurchasePrice != nil {
		purchasePrice = sql.NullFloat64{Float64: *collection.PurchasePrice, Valid: true}
	}
	if collection.Condition != nil {
		condition = sql.NullString{String: string(*collection.Condition), Valid: true}
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE album_collections
		SET notes = $1, date_acquired = $2, purchase_price = $3, condition = $4, updated_at = $5
		WHERE id = $6 AND user_id = $7`,
		notes, dateAcquired, purchasePrice, condition, time.Now().UTC(), id, userID)

	if err != nil {
		return nil, fmt.Errorf("update collection item: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return nil, ErrCollectionNotFound
	}

	// Return updated collection item
	var updated models.AlbumCollection
	err = s.db.QueryRowContext(ctx, `
		SELECT id, user_id, album_id, collection_type, COALESCE(notes, ''), date_added, date_acquired, purchase_price, condition, created_at, updated_at
		FROM album_collections
		WHERE id = $1`, id).Scan(
		&updated.ID, &updated.UserID, &updated.AlbumID, &updated.CollectionType,
		&notes, &updated.DateAdded, &dateAcquired, &purchasePrice, &condition,
		&updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reload collection item: %w", err)
	}

	updated.Notes = notes.String
	if dateAcquired.Valid {
		updated.DateAcquired = &dateAcquired.Time
	}
	if purchasePrice.Valid {
		updated.PurchasePrice = &purchasePrice.Float64
	}
	if condition.Valid {
		cond := models.AlbumCondition(condition.String)
		updated.Condition = &cond
	}

	return &updated, nil
}

// RemoveFromCollection removes an album from a user's collection
func (s *Store) RemoveFromCollection(ctx context.Context, token string, id int64) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM album_collections
		WHERE id = $1 AND user_id = $2`, id, userID)

	if err != nil {
		return fmt.Errorf("delete collection item: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrCollectionNotFound
	}

	return nil
}

// MoveToCollection moves an album from one collection type to another (e.g., wishlist to owned)
func (s *Store) MoveToCollection(ctx context.Context, token string, id int64, targetType models.CollectionType) error {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return err
	}

	// Validate target collection type
	if targetType != models.CollectionTypeWishlist && targetType != models.CollectionTypeOwned {
		return ErrInvalidCollectionType
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE album_collections
		SET collection_type = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4`,
		targetType, time.Now().UTC(), id, userID)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyInCollection
		}
		return fmt.Errorf("move collection item: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrCollectionNotFound
	}

	return nil
}

// GetCollectionStats returns statistics about a user's collection
func (s *Store) GetCollectionStats(ctx context.Context, token string) (*models.CollectionStats, error) {
	userID, err := s.UserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	stats := &models.CollectionStats{
		ByGenre:     make(map[string]int),
		ByCondition: make(map[string]int),
	}

	// Count totals by collection type
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN collection_type = 'wishlist' THEN 1 END) as wishlist_count,
			COUNT(CASE WHEN collection_type = 'owned' THEN 1 END) as owned_count,
			COALESCE(SUM(CASE WHEN collection_type = 'owned' THEN purchase_price ELSE 0 END), 0) as total_value
		FROM album_collections
		WHERE user_id = $1`, userID).Scan(&stats.TotalWishlist, &stats.TotalOwned, &stats.TotalValue)
	if err != nil {
		return nil, fmt.Errorf("get collection totals: %w", err)
	}

	// Get genre breakdown for owned albums
	genreRows, err := s.db.QueryContext(ctx, `
		SELECT a.genre, COUNT(*) as count
		FROM album_collections ac
		JOIN albums a ON ac.album_id = a.id
		WHERE ac.user_id = $1 AND ac.collection_type = 'owned' AND a.genre != ''
		GROUP BY a.genre
		ORDER BY count DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get genre stats: %w", err)
	}
	defer genreRows.Close()

	for genreRows.Next() {
		var genre string
		var count int
		if err := genreRows.Scan(&genre, &count); err != nil {
			return nil, fmt.Errorf("scan genre stat: %w", err)
		}
		stats.ByGenre[genre] = count
	}

	// Get condition breakdown for owned albums
	conditionRows, err := s.db.QueryContext(ctx, `
		SELECT condition, COUNT(*) as count
		FROM album_collections
		WHERE user_id = $1 AND collection_type = 'owned' AND condition IS NOT NULL
		GROUP BY condition
		ORDER BY count DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get condition stats: %w", err)
	}
	defer conditionRows.Close()

	for conditionRows.Next() {
		var condition string
		var count int
		if err := conditionRows.Scan(&condition, &count); err != nil {
			return nil, fmt.Errorf("scan condition stat: %w", err)
		}
		stats.ByCondition[condition] = count
	}

	return stats, nil
}
