package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vinylhound/shared/middleware"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Service URLs
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8001")
	catalogServiceURL := getEnv("CATALOG_SERVICE_URL", "http://localhost:8002")
	ratingServiceURL := getEnv("RATING_SERVICE_URL", "http://localhost:8003")
	playlistServiceURL := getEnv("PLAYLIST_SERVICE_URL", "http://localhost:8004")

	// Create reverse proxies
	userProxy := createProxy(userServiceURL)
	catalogProxy := createProxy(catalogServiceURL)
	ratingProxy := createProxy(ratingServiceURL)
	playlistProxy := createProxy(playlistServiceURL)

	// Setup router
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// User service routes
	api.PathPrefix("/auth").Handler(userProxy)
	api.PathPrefix("/users").Handler(userProxy)

	// Catalog service routes
	api.PathPrefix("/albums").Handler(catalogProxy)
	api.PathPrefix("/artists").Handler(catalogProxy)
	api.PathPrefix("/songs").Handler(catalogProxy)
	api.PathPrefix("/catalog").Handler(catalogProxy)

	// Rating service routes
	api.PathPrefix("/ratings").Handler(ratingProxy)
	api.PathPrefix("/reviews").Handler(ratingProxy)
	api.PathPrefix("/preferences").Handler(ratingProxy)

	// Playlist service routes
	api.PathPrefix("/playlists").Handler(playlistProxy)

	// Start server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Gateway starting on port %s", port)
		log.Printf("User Service: %s", userServiceURL)
		log.Printf("Catalog Service: %s", catalogServiceURL)
		log.Printf("Rating Service: %s", ratingServiceURL)
		log.Printf("Playlist Service: %s", playlistServiceURL)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func createProxy(targetURL string) *httputil.ReverseProxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL %s: %v", targetURL, err)
	}

	return httputil.NewSingleHostReverseProxy(target)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
