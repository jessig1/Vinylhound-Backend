package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"vinylhound/internal/app"
)

func main() {
	_ = godotenv.Load("config/local.env")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env var is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	store := app.NewStore(db)

	if err := store.CreateUser("demo", "demo123", []string{
		"Welcome to Vinylhound!",
		"Start by customizing your personal playlist.",
	}); err != nil && !errors.Is(err, app.ErrUserExists) {
		log.Fatalf("bootstrap demo user: %v", err)
	}

	server := app.NewServer(store)

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	log.Printf("API available at http://localhost%v", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
