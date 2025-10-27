package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vinylhound/rating-service/internal/handlers"
	"vinylhound/rating-service/internal/repository"
	"vinylhound/rating-service/internal/service"
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
	ratingRepo := repository.NewRatingRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	preferenceRepo := repository.NewPreferenceRepository(db)

	// Initialize services
	ratingService := service.NewRatingService(ratingRepo)
	reviewService := service.NewReviewService(reviewRepo)
	preferenceService := service.NewPreferenceService(preferenceRepo)

	// Initialize handlers
	ratingHandler := handlers.NewRatingHandler(ratingService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	preferenceHandler := handlers.NewPreferenceHandler(preferenceService)

	// Setup routes
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public routes
	api.HandleFunc("/ratings", ratingHandler.ListRatings).Methods("GET")
	api.HandleFunc("/ratings/{id}", ratingHandler.GetRating).Methods("GET")
	api.HandleFunc("/reviews", reviewHandler.ListReviews).Methods("GET")
	api.HandleFunc("/reviews/{id}", reviewHandler.GetReview).Methods("GET")

	// Protected routes (require authentication)
	protected := api.PathPrefix("/ratings").Subrouter()
	// Note: In a real microservice, you'd validate tokens with the user service
	// For now, we'll skip auth middleware for simplicity
	protected.HandleFunc("", ratingHandler.CreateRating).Methods("POST")
	protected.HandleFunc("/{id}", ratingHandler.UpdateRating).Methods("PUT")
	protected.HandleFunc("/{id}", ratingHandler.DeleteRating).Methods("DELETE")

	protectedReviews := api.PathPrefix("/reviews").Subrouter()
	protectedReviews.HandleFunc("", reviewHandler.CreateReview).Methods("POST")
	protectedReviews.HandleFunc("/{id}", reviewHandler.UpdateReview).Methods("PUT")
	protectedReviews.HandleFunc("/{id}", reviewHandler.DeleteReview).Methods("DELETE")

	protectedPrefs := api.PathPrefix("/preferences").Subrouter()
	protectedPrefs.HandleFunc("", preferenceHandler.GetPreferences).Methods("GET")
	protectedPrefs.HandleFunc("", preferenceHandler.UpdatePreferences).Methods("PUT")

	// Album preferences
	api.HandleFunc("/me/albums/{id}/preference", preferenceHandler.SetAlbumPreference).Methods("PUT")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Rating service starting on port %s", port)
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
