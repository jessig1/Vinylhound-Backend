package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"vinylhound/shared/go/models"
)

// handleCollections handles GET (list) and POST (add) for collections
func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listCollections(w, r)
	case http.MethodPost:
		s.addToCollection(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCollection handles GET/PUT/DELETE for a specific collection item
func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/collections/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	collectionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	// Check for sub-resources
	if len(parts) >= 2 {
		if parts[1] == "move" && r.Method == http.MethodPost {
			s.moveCollection(w, r, collectionID)
			return
		}
		http.Error(w, "Unknown sub-resource", http.StatusNotFound)
		return
	}

	// Handle collection CRUD
	switch r.Method {
	case http.MethodGet:
		s.getCollection(w, r, collectionID)
	case http.MethodPut:
		s.updateCollection(w, r, collectionID)
	case http.MethodDelete:
		s.removeFromCollection(w, r, collectionID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCollectionStats handles GET for collection statistics
func (s *Server) handleCollectionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	stats, err := s.collections.GetStats(r.Context(), token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters for filtering
	var filter models.CollectionFilter

	if collectionType := r.URL.Query().Get("type"); collectionType != "" {
		ct := models.CollectionType(collectionType)
		filter.CollectionType = &ct
	}

	if artist := r.URL.Query().Get("artist"); artist != "" {
		filter.Artist = artist
	}

	if genre := r.URL.Query().Get("genre"); genre != "" {
		filter.Genre = genre
	}

	if yearFrom := r.URL.Query().Get("year_from"); yearFrom != "" {
		if yf, err := strconv.Atoi(yearFrom); err == nil {
			filter.YearFrom = &yf
		}
	}

	if yearTo := r.URL.Query().Get("year_to"); yearTo != "" {
		if yt, err := strconv.Atoi(yearTo); err == nil {
			filter.YearTo = &yt
		}
	}

	if condition := r.URL.Query().Get("condition"); condition != "" {
		cond := models.AlbumCondition(condition)
		filter.Condition = &cond
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.SearchTerm = search
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	collections, err := s.collections.List(r.Context(), token, filter)
	if err != nil {
		if err.Error() == "unauthorized" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Collections []*models.AlbumCollectionWithDetails `json:"collections"`
		Count       int                                  `json:"count"`
	}{
		Collections: collections,
		Count:       len(collections),
	})
}

func (s *Server) getCollection(w http.ResponseWriter, r *http.Request, id int64) {
	collection, err := s.collections.Get(r.Context(), id)
	if err != nil {
		if err.Error() == "collection item not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

func (s *Server) addToCollection(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		AlbumID        int64                   `json:"album_id"`
		CollectionType models.CollectionType   `json:"collection_type"`
		Notes          string                  `json:"notes,omitempty"`
		DateAcquired   *string                 `json:"date_acquired,omitempty"`
		PurchasePrice  *float64                `json:"purchase_price,omitempty"`
		Condition      *models.AlbumCondition  `json:"condition,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AlbumID == 0 {
		http.Error(w, "album_id is required", http.StatusBadRequest)
		return
	}

	if req.CollectionType != models.CollectionTypeWishlist && req.CollectionType != models.CollectionTypeOwned {
		http.Error(w, "collection_type must be 'wishlist' or 'owned'", http.StatusBadRequest)
		return
	}

	collection := &models.AlbumCollection{
		AlbumID:        req.AlbumID,
		CollectionType: req.CollectionType,
		Notes:          req.Notes,
		PurchasePrice:  req.PurchasePrice,
		Condition:      req.Condition,
	}

	// Parse date_acquired if provided
	if req.DateAcquired != nil && *req.DateAcquired != "" {
		// You might want to parse this properly based on your date format
		// For now, we'll leave it as nil if not properly formatted
		// In production, you'd use time.Parse with the appropriate layout
	}

	created, err := s.collections.Add(r.Context(), token, collection)
	if err != nil {
		if err.Error() == "album already in this collection" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err.Error() == "album not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) updateCollection(w http.ResponseWriter, r *http.Request, id int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Notes         string                  `json:"notes,omitempty"`
		DateAcquired  *string                 `json:"date_acquired,omitempty"`
		PurchasePrice *float64                `json:"purchase_price,omitempty"`
		Condition     *models.AlbumCondition  `json:"condition,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection := &models.AlbumCollection{
		Notes:         req.Notes,
		PurchasePrice: req.PurchasePrice,
		Condition:     req.Condition,
	}

	// Parse date_acquired if provided
	if req.DateAcquired != nil && *req.DateAcquired != "" {
		// Parse date properly in production
	}

	updated, err := s.collections.Update(r.Context(), token, id, collection)
	if err != nil {
		if err.Error() == "collection item not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err.Error() == "not authorized to modify this collection item" {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) removeFromCollection(w http.ResponseWriter, r *http.Request, id int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	err := s.collections.Remove(r.Context(), token, id)
	if err != nil {
		if err.Error() == "collection item not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) moveCollection(w http.ResponseWriter, r *http.Request, id int64) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var req struct {
		TargetType models.CollectionType `json:"target_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TargetType != models.CollectionTypeWishlist && req.TargetType != models.CollectionTypeOwned {
		http.Error(w, "target_type must be 'wishlist' or 'owned'", http.StatusBadRequest)
		return
	}

	err := s.collections.Move(r.Context(), token, id, req.TargetType)
	if err != nil {
		if err.Error() == "collection item not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err.Error() == "album already in this collection" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
