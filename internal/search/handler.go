package search

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Handler responds to search requests backed by the Store.
type Handler struct {
	store Store
}

// NewHandler builds a handler using the provided store implementation.
func NewHandler(store Store) http.Handler {
	return &Handler{store: store}
}

// Response models the payload returned by the search handler.
type Response struct {
	Sections []Section `json:"sections"`
}

// Section groups related search results.
type Section struct {
	Name  string `json:"name"`
	Items []Item `json:"items"`
}

// Item represents a single search result entry.
type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	Description string `json:"description,omitempty"`
	Href        string `json:"href,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, Response{Sections: []Section{}})
		return
	}

	limit := 10
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.store.Search(r.Context(), query, limit)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, buildResponse(results))
}

func buildResponse(results Results) Response {
	var sections []Section

	if len(results.Artists) > 0 {
		items := make([]Item, 0, len(results.Artists))
		for _, artist := range results.Artists {
			subtitle := pluralize(artist.AlbumCount, "album")
			items = append(items, Item{
				ID:        artist.ID,
				Title:     artist.Name,
				Subtitle:  subtitle,
				Href:      artist.Href,
				Thumbnail: artist.ImageURL,
			})
		}
		sections = append(sections, Section{Name: "artists", Items: items})
	}

	if len(results.Albums) > 0 {
		items := make([]Item, 0, len(results.Albums))
		for _, album := range results.Albums {
			subtitle := album.Artist
			if album.ReleaseYear > 0 {
				subtitle = subtitle + " • " + strconv.Itoa(album.ReleaseYear)
			}
			items = append(items, Item{
				ID:        strconv.FormatInt(album.ID, 10),
				Title:     album.Title,
				Subtitle:  subtitle,
				Href:      album.Href,
				Thumbnail: album.ImageURL,
			})
		}
		sections = append(sections, Section{Name: "albums", Items: items})
	}

	if len(results.Songs) > 0 {
		items := make([]Item, 0, len(results.Songs))
		for _, song := range results.Songs {
			subtitle := song.Artist
			if song.Album != "" {
				subtitle = subtitle + " — " + song.Album
			}
			items = append(items, Item{
				ID:        strconv.FormatInt(song.ID, 10),
				Title:     song.Title,
				Subtitle:  subtitle,
				Href:      song.Href,
				Thumbnail: song.ImageURL,
			})
		}
		sections = append(sections, Section{Name: "songs", Items: items})
	}

	return Response{Sections: sections}
}

func writeJSON(w http.ResponseWriter, status int, payload Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func pluralize(count int, singular string) string {
	switch count {
	case 0:
		return ""
	case 1:
		return "1 " + singular
	default:
		return strconv.Itoa(count) + " " + singular + "s"
	}
}
