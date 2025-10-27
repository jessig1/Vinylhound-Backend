package service

import (
	"context"
	"fmt"
	"os"

	"vinylhound/shared/auth"
	"vinylhound/shared/models"
	"vinylhound/user-service/internal/repository"
)

// UserService handles user-related business logic
type UserService struct {
	repo     repository.UserRepository
	tokenMgr *auth.TokenManager
}

// NewUserService creates a new user service
func NewUserService(repo repository.UserRepository) *UserService {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}

	return &UserService{
		repo:     repo,
		tokenMgr: auth.NewTokenManager(jwtSecret),
	}
}

// Signup creates a new user account
func (s *UserService) Signup(ctx context.Context, username, password string, content []string) error {
	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Create user
	userID, err := s.repo.CreateUser(ctx, username, hashedPassword)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	// Add initial content if provided
	if len(content) > 0 {
		if err := s.repo.CreateUserContent(ctx, userID, content); err != nil {
			return fmt.Errorf("create user content: %w", err)
		}
	}

	return nil
}

// Login authenticates a user and returns a session token
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	// Get user by username
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	// Verify password
	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}

	// Generate session token
	token, err := s.tokenMgr.GenerateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	// Create session
	if err := s.repo.CreateSession(ctx, token, user.ID, auth.TokenExpiry()); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

// ValidateToken validates a session token and returns user ID
func (s *UserService) ValidateToken(ctx context.Context, token string) (int64, error) {
	userID, err := s.repo.GetUserIDByToken(ctx, token)
	if err != nil {
		return 0, fmt.Errorf("validate token: %w", err)
	}
	return userID, nil
}

// GetProfile returns user profile information
func (s *UserService) GetProfile(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// GetContent returns user's content
func (s *UserService) GetContent(ctx context.Context, userID int64) ([]string, error) {
	content, err := s.repo.GetUserContent(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get content: %w", err)
	}
	return content, nil
}

// UpdateContent updates user's content
func (s *UserService) UpdateContent(ctx context.Context, userID int64, content []string) error {
	if err := s.repo.UpdateUserContent(ctx, userID, content); err != nil {
		return fmt.Errorf("update content: %w", err)
	}
	return nil
}
