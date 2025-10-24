package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/rating-service/internal/service"
	"vinylhound/shared/models"

	"github.com/gorilla/mux"
)

// RatingHandler handles HTTP requests for rating operations
type RatingHandler struct {
	ratingService *service.RatingService
}

// NewRatingHandler creates a new rating handler
func NewRatingHandler(ratingService *service.RatingService) *RatingHandler {
	return &RatingHandler{ratingService: ratingService}
}

// ListRatings handles listing ratings with optional filtering
func (h *RatingHandler) ListRatings(w http.ResponseWriter, r *http.Request) {
	filter := models.RatingFilter{}

	// Parse query parameters
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		if id, err := strconv.ParseInt(userID, 10, 64); err == nil {
			filter.UserID = id
		}
	}

	if albumID := r.URL.Query().Get("album_id"); albumID != "" {
		if id, err := strconv.ParseInt(albumID, 10, 64); err == nil {
			filter.AlbumID = id
		}
	}

	if minRating := r.URL.Query().Get("min_rating"); minRating != "" {
		if rating, err := strconv.Atoi(minRating); err == nil {
			filter.MinRating = rating
		}
	}

	if maxRating := r.URL.Query().Get("max_rating"); maxRating != "" {
		if rating, err := strconv.Atoi(maxRating); err == nil {
			filter.MaxRating = rating
		}
	}

	// Parse pagination
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

	ratings, err := h.ratingService.ListRatings(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string][]*models.Rating{"ratings": ratings})
}

// GetRating handles getting a specific rating
func (h *RatingHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid rating ID", http.StatusBadRequest)
		return
	}

	rating, err := h.ratingService.GetRating(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(rating)
}

// CreateRating handles creating a new rating
func (h *RatingHandler) CreateRating(w http.ResponseWriter, r *http.Request) {
	var rating models.Rating
	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if rating.UserID == 0 || rating.AlbumID == 0 || rating.Rating == 0 {
		http.Error(w, "User ID, album ID, and rating are required", http.StatusBadRequest)
		return
	}

	createdRating, err := h.ratingService.CreateRating(r.Context(), &rating)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdRating)
}

// UpdateRating handles updating an existing rating
func (h *RatingHandler) UpdateRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid rating ID", http.StatusBadRequest)
		return
	}

	var rating models.Rating
	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedRating, err := h.ratingService.UpdateRating(r.Context(), id, &rating)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(updatedRating)
}

// DeleteRating handles deleting a rating
func (h *RatingHandler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid rating ID", http.StatusBadRequest)
		return
	}

	if err := h.ratingService.DeleteRating(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
