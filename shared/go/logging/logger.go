package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// contextKey is the type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey contextKey = "request_id"
	// UserIDKey is the context key for user IDs
	UserIDKey contextKey = "user_id"
)

// Logger wraps zerolog for application logging
type Logger struct {
	logger zerolog.Logger
}

// Config holds logging configuration
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output io.Writer
}

// New creates a new logger with the given configuration
func New(cfg Config) *Logger {
	// Set output
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Create logger based on format
	var logger zerolog.Logger
	if cfg.Format == "text" {
		// Pretty console output for development
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}).
			Level(level).
			With().
			Timestamp().
			Logger()
	} else {
		// JSON output for production
		logger = zerolog.New(output).
			Level(level).
			With().
			Timestamp().
			Logger()
	}

	return &Logger{logger: logger}
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	log.Logger = logger.logger
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(err error, msg string) {
	l.logger.Fatal().Err(err).Msg(msg)
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *zerolog.Logger {
	logger := l.logger.With()

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.Str("request_id", requestID.(string))
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.Int64("user_id", userID.(int64))
	}

	contextLogger := logger.Logger()
	return &contextLogger
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *zerolog.Logger {
	event := l.logger.With()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	logger := event.Logger()
	return &logger
}

// HTTPRequestLogger logs HTTP request details
func (l *Logger) HTTPRequest(method, path string, statusCode int, duration time.Duration, err error) {
	event := l.logger.Info()
	if statusCode >= 400 {
		event = l.logger.Error()
	}

	event.
		Str("method", method).
		Str("path", path).
		Int("status_code", statusCode).
		Dur("duration_ms", duration).
		Err(err).
		Msg("HTTP request")
}

// DBQuery logs database query details
func (l *Logger) DBQuery(query string, duration time.Duration, err error) {
	event := l.logger.Debug()
	if err != nil {
		event = l.logger.Error()
	}

	event.
		Str("query", query).
		Dur("duration_ms", duration).
		Err(err).
		Msg("Database query")
}

// Global logger functions for convenience

// Debug logs a debug message using the global logger
func Debug(msg string) {
	log.Debug().Msg(msg)
}

// Info logs an info message using the global logger
func Info(msg string) {
	log.Info().Msg(msg)
}

// Warn logs a warning message using the global logger
func Warn(msg string) {
	log.Warn().Msg(msg)
}

// Error logs an error message using the global logger
func Error(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(err error, msg string) {
	log.Fatal().Err(err).Msg(msg)
}

// WithContext returns a logger with context values from the global logger
func WithContext(ctx context.Context) *zerolog.Logger {
	logger := log.With()

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.Str("request_id", requestID.(string))
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.Int64("user_id", userID.(int64))
	}

	contextLogger := logger.Logger()
	return &contextLogger
}
