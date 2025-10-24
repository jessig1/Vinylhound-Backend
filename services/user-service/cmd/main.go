package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vinylhound/shared/database"
	"vinylhound/shared/middleware"
	"vinylhound/user-service/internal/handlers"
	"vinylhound/user-service/internal/repository"
	"vinylhound/user-service/internal/service"

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

	// Initialize repository
	userRepo := repository.NewUserRepository(db)

	// Initialize service
	userService := service.NewUserService(userRepo)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)

	// Setup routes
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Auth routes (no auth required)
	api.HandleFunc("/auth/signup", userHandler.Signup).Methods("POST")
	api.HandleFunc("/auth/login", userHandler.Login).Methods("POST")

	// Protected routes
	protected := api.PathPrefix("/users").Subrouter()
	protected.Use(middleware.AuthMiddleware(userService))
	protected.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/content", userHandler.GetContent).Methods("GET")
	protected.HandleFunc("/content", userHandler.UpdateContent).Methods("PUT")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("User service starting on port %s", port)
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
