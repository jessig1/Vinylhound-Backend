package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains application-wide settings sourced from the environment.
type Config struct {
	DatabaseURL    string
	Addr           string
	AllowedOrigins []string
}

func loadConfig() (Config, error) {
	_ = godotenv.Load("config/local.env")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return Config{}, errors.New("DATABASE_URL env var is required")
	}

	addr := fmt.Sprintf(":%s", envOrDefault("PORT", "8080"))

	origins := parseAllowedOrigins(envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))

	return Config{
		DatabaseURL:    dsn,
		Addr:           addr,
		AllowedOrigins: origins,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	var origins []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
