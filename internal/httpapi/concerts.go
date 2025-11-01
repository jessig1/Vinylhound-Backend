package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"vinylhound/shared/go/models"
)

func (s *Server) handleCreateConcert(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing or invalid token"})
		return
	}

	var req struct {
		VenueID     int64     `json:"venue_id"`
		ArtistName  string    `json:"artist_name"`
		Name        string    `json:"name"`
		Date        time.Time `json:"date"`
		TicketPrice *float64  `json:"ticket_price,omitempty"`
		Notes       string    `json:"notes,omitempty"`
		Attended    bool      `json:"attended"`
		Rating      *int      `json:"rating,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	// Validation
	if req.VenueID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "venue_id is required"})
		return
	}
	if req.ArtistName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "artist_name is required"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}

	concert := &models.Concert{
		VenueID:     req.VenueID,
		ArtistName:  req.ArtistName,
		Name:        req.Name,
		Date:        req.Date,
		TicketPrice: req.TicketPrice,
		Notes:       req.Notes,
		Attended:    req.Attended,
		Rating:      req.Rating,
	}

	created, err := s.concerts.Create(r.Context(), token, concert)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListConcerts(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing or invalid token"})
		return
	}

	// Check for query params to filter
	artistName := r.URL.Query().Get("artist")
	venueIDStr := r.URL.Query().Get("venue_id")
	upcoming := r.URL.Query().Get("upcoming") == "true"

	var concerts []*models.ConcertWithDetails
	var err error

	if artistName != "" {
		concerts, err = s.concerts.ListByArtist(r.Context(), token, artistName)
	} else if venueIDStr != "" {
		venueID, parseErr := strconv.ParseInt(venueIDStr, 10, 64)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid venue_id"})
			return
		}
		concerts, err = s.concerts.ListByVenue(r.Context(), venueID)
	} else if upcoming {
		concerts, err = s.concerts.ListUpcoming(r.Context(), token)
	} else {
		concerts, err = s.concerts.List(r.Context(), token)
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, concerts)
}

func (s *Server) handleGetConcert(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid concert ID"})
		return
	}

	concert, err := s.concerts.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, concert)
}

func (s *Server) handleUpdateConcert(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing or invalid token"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid concert ID"})
		return
	}

	var concert models.Concert
	if err := json.NewDecoder(r.Body).Decode(&concert); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	updated, err := s.concerts.Update(r.Context(), token, id, &concert)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteConcert(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing or invalid token"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid concert ID"})
		return
	}

	if err := s.concerts.Delete(r.Context(), token, id); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkConcertAttended(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing or invalid token"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid concert ID"})
		return
	}

	var req struct {
		Rating *int `json:"rating,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if err := s.concerts.MarkAttended(r.Context(), token, id, req.Rating); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
