package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/rating-service/internal/service"
	"vinylhound/shared/models"

	"github.com/gorilla/mux"
)

// ReviewHandler handles HTTP requests for review operations
type ReviewHandler struct {
	reviewService *service.ReviewService
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(reviewService *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

// ListReviews handles listing reviews with optional filtering
func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	var userID, albumID int64

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = id
		}
	}

	if albumIDStr := r.URL.Query().Get("album_id"); albumIDStr != "" {
		if id, err := strconv.ParseInt(albumIDStr, 10, 64); err == nil {
			albumID = id
		}
	}

	reviews, err := h.reviewService.ListReviews(r.Context(), userID, albumID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string][]*models.Review{"reviews": reviews})
}

// GetReview handles getting a specific review
func (h *ReviewHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}

	review, err := h.reviewService.GetReview(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(review)
}

// CreateReview handles creating a new review
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var review models.Review
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if review.UserID == 0 || review.AlbumID == 0 || review.Title == "" {
		http.Error(w, "User ID, album ID, and title are required", http.StatusBadRequest)
		return
	}

	createdReview, err := h.reviewService.CreateReview(r.Context(), &review)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdReview)
}

// UpdateReview handles updating an existing review
func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}

	var review models.Review
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedReview, err := h.reviewService.UpdateReview(r.Context(), id, &review)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(updatedReview)
}

// DeleteReview handles deleting a review
func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid review ID", http.StatusBadRequest)
		return
	}

	if err := h.reviewService.DeleteReview(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
