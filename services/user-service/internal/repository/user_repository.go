package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vinylhound/shared/models"
)

// userRepository handles user data persistence
type userRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a new user and their default favorites playlist
func (r *userRepository) CreateUser(ctx context.Context, username, passwordHash string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var userID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, username, passwordHash, time.Now(), time.Now()).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	// Create the default favorites playlist
	_, err = tx.ExecContext(ctx, `
		INSERT INTO playlists (title, description, owner, user_id, is_favorite, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, FALSE, $5, $6)
	`, "Favorites", "Your favorited songs and albums", username, userID, time.Now(), time.Now())

	if err != nil {
		return 0, fmt.Errorf("create favorites playlist: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves a user by username
func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*UserWithPassword, error) {
	user := &UserWithPassword{User: &models.User{}}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

// CreateSession creates a new user session
func (r *userRepository) CreateSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`, token, userID, time.Now(), expiresAt)

	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetUserIDByToken retrieves user ID by session token
func (r *userRepository) GetUserIDByToken(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM sessions
		WHERE token = $1 AND expires_at > $2
	`, token, time.Now()).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("invalid or expired token")
		}
		return 0, fmt.Errorf("get user by token: %w", err)
	}

	return userID, nil
}

// CreateUserContent creates user content entries
func (r *userRepository) CreateUserContent(ctx context.Context, userID int64, content []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, entry := range content {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO user_content (user_id, position, entry, created_at)
			VALUES ($1, $2, $3, $4)
		`, userID, i, entry, time.Now())
		if err != nil {
			return fmt.Errorf("insert content: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetUserContent retrieves user content
func (r *userRepository) GetUserContent(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT entry
		FROM user_content
		WHERE user_id = $1
		ORDER BY position ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query content: %w", err)
	}
	defer rows.Close()

	var content []string
	for rows.Next() {
		var entry string
		if err := rows.Scan(&entry); err != nil {
			return nil, fmt.Errorf("scan content: %w", err)
		}
		content = append(content, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content: %w", err)
	}

	return content, nil
}

// UpdateUserContent updates user content
func (r *userRepository) UpdateUserContent(ctx context.Context, userID int64, content []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing content
	_, err = tx.ExecContext(ctx, `
		DELETE FROM user_content
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("delete content: %w", err)
	}

	// Insert new content
	for i, entry := range content {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO user_content (user_id, position, entry, created_at)
			VALUES ($1, $2, $3, $4)
		`, userID, i, entry, time.Now())
		if err != nil {
			return fmt.Errorf("insert content: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
