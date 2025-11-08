package collections

import (
	"context"

	"vinylhound/shared/go/models"
)

// Store defines persistence operations for album collections
type Store interface {
	AddToCollection(ctx context.Context, token string, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	ListCollection(ctx context.Context, token string, filter models.CollectionFilter) ([]*models.AlbumCollectionWithDetails, error)
	GetCollectionItem(ctx context.Context, id int64) (*models.AlbumCollectionWithDetails, error)
	UpdateCollectionItem(ctx context.Context, token string, id int64, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	RemoveFromCollection(ctx context.Context, token string, id int64) error
	MoveToCollection(ctx context.Context, token string, id int64, targetType models.CollectionType) error
	GetCollectionStats(ctx context.Context, token string) (*models.CollectionStats, error)
}

// Service coordinates collection-related operations
type Service interface {
	Add(ctx context.Context, token string, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	List(ctx context.Context, token string, filter models.CollectionFilter) ([]*models.AlbumCollectionWithDetails, error)
	Get(ctx context.Context, id int64) (*models.AlbumCollectionWithDetails, error)
	Update(ctx context.Context, token string, id int64, collection *models.AlbumCollection) (*models.AlbumCollection, error)
	Remove(ctx context.Context, token string, id int64) error
	Move(ctx context.Context, token string, id int64, targetType models.CollectionType) error
	GetStats(ctx context.Context, token string) (*models.CollectionStats, error)

	// Convenience methods for specific collection types
	AddToWishlist(ctx context.Context, token string, albumID int64, notes string) (*models.AlbumCollection, error)
	AddToOwned(ctx context.Context, token string, albumID int64, notes string, dateAcquired *models.AlbumCollection) (*models.AlbumCollection, error)
	MoveToOwned(ctx context.Context, token string, collectionID int64) error
}

type service struct {
	store Store
}

// New constructs a collections Service
func New(store Store) Service {
	return &service{
		store: store,
	}
}

func (s *service) Add(ctx context.Context, token string, collection *models.AlbumCollection) (*models.AlbumCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.AddToCollection(ctx, token, collection)
}

func (s *service) List(ctx context.Context, token string, filter models.CollectionFilter) ([]*models.AlbumCollectionWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ListCollection(ctx, token, filter)
}

func (s *service) Get(ctx context.Context, id int64) (*models.AlbumCollectionWithDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetCollectionItem(ctx, id)
}

func (s *service) Update(ctx context.Context, token string, id int64, collection *models.AlbumCollection) (*models.AlbumCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.UpdateCollectionItem(ctx, token, id, collection)
}

func (s *service) Remove(ctx context.Context, token string, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.RemoveFromCollection(ctx, token, id)
}

func (s *service) Move(ctx context.Context, token string, id int64, targetType models.CollectionType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.MoveToCollection(ctx, token, id, targetType)
}

func (s *service) GetStats(ctx context.Context, token string) (*models.CollectionStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.GetCollectionStats(ctx, token)
}

// AddToWishlist is a convenience method to add an album to wishlist
func (s *service) AddToWishlist(ctx context.Context, token string, albumID int64, notes string) (*models.AlbumCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	collection := &models.AlbumCollection{
		AlbumID:        albumID,
		CollectionType: models.CollectionTypeWishlist,
		Notes:          notes,
	}

	return s.store.AddToCollection(ctx, token, collection)
}

// AddToOwned is a convenience method to add an album to owned collection
func (s *service) AddToOwned(ctx context.Context, token string, albumID int64, notes string, details *models.AlbumCollection) (*models.AlbumCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	collection := &models.AlbumCollection{
		AlbumID:        albumID,
		CollectionType: models.CollectionTypeOwned,
		Notes:          notes,
	}

	// Copy optional details if provided
	if details != nil {
		collection.DateAcquired = details.DateAcquired
		collection.PurchasePrice = details.PurchasePrice
		collection.Condition = details.Condition
	}

	return s.store.AddToCollection(ctx, token, collection)
}

// MoveToOwned is a convenience method to move an album from wishlist to owned
func (s *service) MoveToOwned(ctx context.Context, token string, collectionID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.MoveToCollection(ctx, token, collectionID, models.CollectionTypeOwned)
}
