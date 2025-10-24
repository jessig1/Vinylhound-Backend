package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// openDatabase establishes a database connection and retries until the instance responds.
func openDatabase(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	const (
		pingTimeout    = 5 * time.Second
		maxWait        = 30 * time.Second
		initialBackoff = 500 * time.Millisecond
		maxBackoff     = 5 * time.Second
	)

	deadline := time.Now().Add(maxWait)
	backoff := initialBackoff
	var lastErr error

	for {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		lastErr = db.PingContext(pingCtx)
		cancel()

		if lastErr == nil {
			return db, nil
		}

		// Respect caller cancellation.
		if ctx.Err() != nil {
			break
		}

		if time.Now().After(deadline) {
			break
		}

		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	_ = db.Close()
	return nil, fmt.Errorf("ping database: %w", lastErr)
}
