package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		log.Fatal("Usage: go run migrate.go [up|down]")
	}

	// Database connection parameters
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	// Connect to database
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create the postgres driver for migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create postgres driver: %v", err)
	}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Find migrations directory (2 levels up from cmd/migrate)
	migrationsPath := filepath.Join(filepath.Dir(filepath.Dir(wd)), "migrations")

	// Convert to absolute path and use filepath.ToSlash for cross-platform compatibility
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(absPath))

	// Create a new migrate instance
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Run migrations
	if os.Args[1] == "up" {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Warning: migration error: %v", err)
			// Try to force clean state and retry
			m.Force(1)
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("Failed to run migrations after force: %v", err)
			}
		}
		log.Println("Migrations applied successfully")
	} else {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	}
}
