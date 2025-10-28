package main

import (
	"net/http"
	"strings"

	"vinylhound/internal/app/albums"
	"vinylhound/internal/app/artists"
	"vinylhound/internal/app/playlists"
	"vinylhound/internal/app/ratings"
	"vinylhound/internal/app/songs"
	"vinylhound/internal/app/users"
	"vinylhound/internal/httpapi"
	"vinylhound/internal/store"
)

func newHTTPHandler(cfg Config, dataStore *store.Store) http.Handler {
	userSvc := users.New(dataStore)
	albumSvc := albums.New(dataStore)
	ratingsSvc := ratings.New(dataStore)
	artistSvc := artists.New(albumSvc)
	songSvc := songs.New(albumSvc, dataStore)
	playlistSvc := playlists.New(dataStore)

	return withCORS(cfg.AllowedOrigins, httpapi.New(userSvc, artistSvc, albumSvc, songSvc, ratingsSvc, playlistSvc).Routes())
}

func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	originAllowed := func(origin string) bool {
		if len(allowedOrigins) == 0 || origin == "" {
			return false
		}
		for _, o := range allowedOrigins {
			if strings.EqualFold(o, origin) {
				return true
			}
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
