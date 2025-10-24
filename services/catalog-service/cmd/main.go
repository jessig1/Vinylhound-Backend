package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vinylhound/catalog-service/internal/handlers"
	"vinylhound/catalog-service/internal/repository"
	"vinylhound/catalog-service/internal/service"
	"vinylhound/shared/database"
	"vinylhound/shared/middleware"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Connect to database
	db, err := database.ConnectFromEnv()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	albumRepo := repository.NewAlbumRepository(db)
	artistRepo := repository.NewArtistRepository(db)
	songRepo := repository.NewSongRepository(db)

	// Initialize services
	albumService := service.NewAlbumService(albumRepo)
	artistService := service.NewArtistService(artistRepo)
	songService := service.NewSongService(songRepo)

	// Initialize handlers
	albumHandler := handlers.NewAlbumHandler(albumService)
	artistHandler := handlers.NewArtistHandler(artistService)
	songHandler := handlers.NewSongHandler(songService)

	// Setup routes
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public routes
	api.HandleFunc("/albums", albumHandler.ListAlbums).Methods("GET")
	api.HandleFunc("/albums/{id}", albumHandler.GetAlbum).Methods("GET")
	api.HandleFunc("/artists", artistHandler.ListArtists).Methods("GET")
	api.HandleFunc("/artists/{id}", artistHandler.GetArtist).Methods("GET")
	api.HandleFunc("/songs", songHandler.ListSongs).Methods("GET")
	api.HandleFunc("/songs/{id}", songHandler.GetSong).Methods("GET")

	// Protected routes (require authentication)
	protected := api.PathPrefix("/catalog").Subrouter()
	// Note: In a real microservice, you'd validate tokens with the user service
	// For now, we'll skip auth middleware for simplicity
	protected.HandleFunc("/albums", albumHandler.CreateAlbum).Methods("POST")
	protected.HandleFunc("/albums/{id}", albumHandler.UpdateAlbum).Methods("PUT")
	protected.HandleFunc("/albums/{id}", albumHandler.DeleteAlbum).Methods("DELETE")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Catalog service starting on port %s", port)
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
