package repository

import (
	"context"
	"time"

	"vinylhound/shared/models"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	CreateUser(ctx context.Context, username, passwordHash string) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*UserWithPassword, error)
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
	CreateSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error
	GetUserIDByToken(ctx context.Context, token string) (int64, error)
	CreateUserContent(ctx context.Context, userID int64, content []string) error
	GetUserContent(ctx context.Context, userID int64) ([]string, error)
	UpdateUserContent(ctx context.Context, userID int64, content []string) error
}

// UserWithPassword represents a user with password hash
type UserWithPassword struct {
	*models.User
	PasswordHash string
}
