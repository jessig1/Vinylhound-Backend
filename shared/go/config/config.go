package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Database configuration
	Database DatabaseConfig

	// Server configuration
	Server ServerConfig

	// Security configuration
	Security SecurityConfig

	// CORS configuration
	CORS CORSConfig

	// Logging configuration
	Logging LoggingConfig
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL      string // Full PostgreSQL URL
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port int
	Host string
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	JWTSecret string
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowedOrigins []string
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Load database configuration
	if err := cfg.loadDatabase(); err != nil {
		return nil, fmt.Errorf("load database config: %w", err)
	}

	// Load server configuration
	if err := cfg.loadServer(); err != nil {
		return nil, fmt.Errorf("load server config: %w", err)
	}

	// Load security configuration
	if err := cfg.loadSecurity(); err != nil {
		return nil, fmt.Errorf("load security config: %w", err)
	}

	// Load CORS configuration
	cfg.loadCORS()

	// Load logging configuration
	cfg.loadLogging()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadDatabase() error {
	// Try to load DATABASE_URL first
	c.Database.URL = os.Getenv("DATABASE_URL")

	// If not present, construct from individual parameters
	if c.Database.URL == "" {
		c.Database.Host = getEnvOrDefault("DB_HOST", "localhost")
		c.Database.User = os.Getenv("DB_USER")
		c.Database.Password = os.Getenv("DB_PASSWORD")
		c.Database.Name = os.Getenv("DB_NAME")
		c.Database.SSLMode = getEnvOrDefault("DB_SSLMODE", "disable")

		portStr := os.Getenv("DB_PORT")
		if portStr == "" {
			if c.Database.Host == "localhost" || c.Database.Host == "127.0.0.1" {
				portStr = "54320"
			} else {
				portStr = "5432"
			}
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid DB_PORT: %w", err)
		}
		c.Database.Port = port

		// Construct URL if all components are present
		if c.Database.Host != "" && c.Database.User != "" && c.Database.Name != "" {
			c.Database.URL = fmt.Sprintf(
				"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
				c.Database.User,
				c.Database.Password,
				c.Database.Host,
				c.Database.Port,
				c.Database.Name,
				c.Database.SSLMode,
			)
		}
	}

	return nil
}

func (c *Config) loadServer() error {
	portStr := getEnvOrDefault("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid PORT: %w", err)
	}
	c.Server.Port = port
	c.Server.Host = getEnvOrDefault("HOST", "0.0.0.0")
	return nil
}

func (c *Config) loadSecurity() error {
	c.Security.JWTSecret = os.Getenv("JWT_SECRET")
	return nil
}

func (c *Config) loadCORS() {
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if originsEnv != "" {
		origins := strings.Split(originsEnv, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		c.CORS.AllowedOrigins = origins
	} else {
		// Default for local development
		c.CORS.AllowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
		}
	}
}

func (c *Config) loadLogging() {
	c.Logging.Level = getEnvOrDefault("LOG_LEVEL", "info")
	c.Logging.Format = getEnvOrDefault("LOG_FORMAT", "json")
}

// Validate checks that all required configuration is present and valid
func (c *Config) Validate() error {
	var errors []string

	// Validate database configuration
	if c.Database.URL == "" {
		errors = append(errors, "DATABASE_URL is required (or DB_HOST, DB_USER, DB_NAME)")
	}

	// Validate security configuration
	if c.Security.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET is required")
	}
	if len(c.Security.JWTSecret) < 16 {
		errors = append(errors, "JWT_SECRET must be at least 16 characters")
	}

	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, "PORT must be between 1 and 65535")
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		errors = append(errors, "LOG_LEVEL must be one of: debug, info, warn, error")
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.Logging.Format] {
		errors = append(errors, "LOG_FORMAT must be one of: json, text")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(os.Getenv("ENV"))
	return env == "" || env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	env := strings.ToLower(os.Getenv("ENV"))
	return env == "production"
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
