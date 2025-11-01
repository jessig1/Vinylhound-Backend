package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"vinylhound/internal/app/albums"
	"vinylhound/internal/app/artists"
	"vinylhound/internal/app/concerts"
	"vinylhound/internal/app/favorites"
	"vinylhound/internal/app/places"
	"vinylhound/internal/app/playlists"
	"vinylhound/internal/app/ratings"
	"vinylhound/internal/app/songs"
	"vinylhound/internal/app/users"
	"vinylhound/internal/httpapi"
	"vinylhound/internal/musicapi"
	"vinylhound/internal/searchservice"
	"vinylhound/internal/store"
)

func newHTTPHandler(cfg Config, db *sql.DB, dataStore *store.Store) http.Handler {
	// Base services
	userSvc := users.New(dataStore)
	albumSvc := albums.New(dataStore)
	ratingsSvc := ratings.New(dataStore)
	playlistSvc := playlists.New(dataStore)
	favoritesSvc := favorites.New(dataStore)

	// Derived services
	artistSvc := artists.New(albumSvc)
	songSvc := songs.New(albumSvc, dataStore)
	searchSvc := newSearchService(cfg, db, dataStore)

	// Place services
	placesSvc := places.New(dataStore)

	// Concert service (depends on places service)
	concertsSvc := concerts.New(dataStore, placesSvc)

	return withCORS(cfg.AllowedOrigins, httpapi.New(userSvc, artistSvc, albumSvc, songSvc, ratingsSvc, playlistSvc, favoritesSvc, searchSvc, placesSvc, concertsSvc).Routes())
}

func newSearchService(cfg Config, db *sql.DB, dataStore *store.Store) *searchservice.Service {
	var spotifyClient musicapi.MusicAPIClient

	// Initialize Spotify client if credentials are provided
	if cfg.SpotifyClientID != "" && cfg.SpotifyClientSecret != "" {
		spotifyClient = musicapi.NewSpotifyClient(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
		log.Println("Spotify client initialized")
	} else {
		log.Println("Spotify credentials not provided, Spotify search disabled")
	}

	return searchservice.NewService(db, spotifyClient, nil, dataStore)
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
