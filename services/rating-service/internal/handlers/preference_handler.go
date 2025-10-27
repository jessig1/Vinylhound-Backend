package handlers

import (
	"encoding/json"
	"net/http"

	"vinylhound/rating-service/internal/service"
	"vinylhound/shared/middleware"
	"vinylhound/shared/models"
)

// PreferenceHandler handles HTTP requests for preference operations
type PreferenceHandler struct {
	preferenceService *service.PreferenceService
}

// NewPreferenceHandler creates a new preference handler
func NewPreferenceHandler(preferenceService *service.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{preferenceService: preferenceService}
}

// GetPreferences handles getting user preferences
func (h *PreferenceHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	preferences, err := h.preferenceService.GetPreferences(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string][]*models.GenrePreference{"preferences": preferences})
}

// UpdatePreferencesRequest represents an update preferences request
type UpdatePreferencesRequest struct {
	Preferences []*models.GenrePreference `json:"preferences"`
}

// UpdatePreferences handles updating user preferences
func (h *PreferenceHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.preferenceService.UpdatePreferences(r.Context(), userID, req.Preferences); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Preferences updated successfully"})
}

// SetAlbumPreference handles setting an album preference for a user
// TODO: This should be moved to a separate AlbumPreferenceHandler/Service/Repository
// as it deals with album preferences (UserPreference model) not genre preferences (GenrePreference model)
func (h *PreferenceHandler) SetAlbumPreference(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented - album preferences should use a separate endpoint", http.StatusNotImplemented)
}
