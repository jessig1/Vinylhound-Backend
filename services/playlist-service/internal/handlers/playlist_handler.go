package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vinylhound/playlist-service/internal/repository"
	"vinylhound/playlist-service/internal/service"
	"vinylhound/shared/models"

	"github.com/gorilla/mux"
)

// PlaylistHandler wires HTTP endpoints to the playlist service.
type PlaylistHandler struct {
	svc *service.PlaylistService
}

// New creates a PlaylistHandler.
func New(svc *service.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{svc: svc}
}

// Register mounts playlist routes on the given router.
func (h *PlaylistHandler) Register(router *mux.Router) {
	router.HandleFunc("/playlists", h.list).Methods(http.MethodGet)
	router.HandleFunc("/playlists", h.create).Methods(http.MethodPost)
	router.HandleFunc("/playlists/{id}", h.get).Methods(http.MethodGet)
	router.HandleFunc("/playlists/{id}", h.update).Methods(http.MethodPut)
	router.HandleFunc("/playlists/{id}", h.delete).Methods(http.MethodDelete)
}

func (h *PlaylistHandler) list(w http.ResponseWriter, r *http.Request) {
	playlists, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]*models.Playlist{"playlists": playlists})
}

func (h *PlaylistHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	playlist, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if err == repository.ErrPlaylistNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, playlist)
}

func (h *PlaylistHandler) create(w http.ResponseWriter, r *http.Request) {
	var payload models.Playlist
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Create(r.Context(), &payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *PlaylistHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	var payload models.Playlist
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Update(r.Context(), id, &payload)
	if err != nil {
		if err == repository.ErrPlaylistNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *PlaylistHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if err == repository.ErrPlaylistNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		http.Error(w, "missing playlist id", http.StatusBadRequest)
		return 0, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid playlist id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
