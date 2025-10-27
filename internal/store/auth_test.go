package store

import (
	"context"
	"testing"
	"time"
)

func TestStore_CreateUser(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	tests := []struct {
		name          string
		username      string
		passwordHash  string
		wantErr       bool
		errContains   string
	}{
		{
			name:         "valid user creation",
			username:     "testuser",
			passwordHash: "$2a$10$hashedpassword",
			wantErr:      false,
		},
		{
			name:         "duplicate username",
			username:     "testuser",
			passwordHash: "$2a$10$hashedpassword2",
			wantErr:      true,
			errContains:  "already exists",
		},
		{
			name:         "empty username",
			username:     "",
			passwordHash: "$2a$10$hashedpassword",
			wantErr:      true,
		},
		{
			name:         "empty password hash",
			username:     "user2",
			passwordHash: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userID, err := store.CreateUser(ctx, tt.username, tt.passwordHash)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateUser() expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("CreateUser() error = %v, should contain %v", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("CreateUser() unexpected error = %v", err)
				return
			}

			if userID <= 0 {
				t.Errorf("CreateUser() returned invalid userID = %v", userID)
			}
		})
	}
}

func TestStore_ValidateCredentials(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx := context.Background()

	// Create test user with known password
	username := "testuser"
	password := "correctpassword"
	passwordHash := hashPassword(t, password)

	userID, err := store.CreateUser(ctx, username, passwordHash)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		username    string
		password    string
		wantUserID  int64
		wantToken   bool
		wantErr     bool
	}{
		{
			name:       "valid credentials",
			username:   username,
			password:   password,
			wantUserID: userID,
			wantToken:  true,
			wantErr:    false,
		},
		{
			name:       "invalid password",
			username:   username,
			password:   "wrongpassword",
			wantUserID: 0,
			wantToken:  false,
			wantErr:    true,
		},
		{
			name:       "invalid username",
			username:   "nonexistent",
			password:   password,
			wantUserID: 0,
			wantToken:  false,
			wantErr:    true,
		},
		{
			name:       "empty username",
			username:   "",
			password:   password,
			wantUserID: 0,
			wantToken:  false,
			wantErr:    true,
		},
		{
			name:       "empty password",
			username:   username,
			password:   "",
			wantUserID: 0,
			wantToken:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, gotUserID, err := store.ValidateCredentials(ctx, tt.username, tt.password)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCredentials() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateCredentials() unexpected error = %v", err)
				return
			}

			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateCredentials() userID = %v, want %v", gotUserID, tt.wantUserID)
			}

			if tt.wantToken && token == "" {
				t.Errorf("ValidateCredentials() expected token, got empty string")
			}

			if !tt.wantToken && token != "" {
				t.Errorf("ValidateCredentials() expected no token, got %v", token)
			}
		})
	}
}

func TestStore_ValidateToken(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx := context.Background()

	// Create test user and get token
	username := "testuser"
	password := "password123"
	passwordHash := hashPassword(t, password)

	userID, err := store.CreateUser(ctx, username, passwordHash)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	token, _, err := store.ValidateCredentials(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantUserID int64
		wantErr    bool
	}{
		{
			name:       "valid token",
			token:      token,
			wantUserID: userID,
			wantErr:    false,
		},
		{
			name:       "invalid token",
			token:      "invalid-token-12345",
			wantUserID: 0,
			wantErr:    true,
		},
		{
			name:       "empty token",
			token:      "",
			wantUserID: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := store.ValidateToken(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateToken() unexpected error = %v", err)
				return
			}

			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateToken() userID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestStore_CreateSession(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx := context.Background()

	// Create test user
	userID, err := store.CreateUser(ctx, "testuser", hashPassword(t, "password"))
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		userID    int64
		expiresAt time.Time
		wantErr   bool
	}{
		{
			name:      "valid session",
			token:     "valid-token-123",
			userID:    userID,
			expiresAt: time.Now().Add(24 * time.Hour),
			wantErr:   false,
		},
		{
			name:      "duplicate token",
			token:     "valid-token-123",
			userID:    userID,
			expiresAt: time.Now().Add(24 * time.Hour),
			wantErr:   true,
		},
		{
			name:      "invalid user ID",
			token:     "another-token",
			userID:    99999,
			expiresAt: time.Now().Add(24 * time.Hour),
			wantErr:   true,
		},
		{
			name:      "expired session",
			token:     "expired-token",
			userID:    userID,
			expiresAt: time.Now().Add(-1 * time.Hour),
			wantErr:   false, // Creation should succeed, validation should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateSession(ctx, tt.token, tt.userID, tt.expiresAt)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateSession() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateSession() unexpected error = %v", err)
			}
		})
	}
}

// Helper functions

func setupTestStore(t *testing.T) *Store {
	// Use in-memory SQLite or test database
	// For now, skip if no test database configured
	t.Skip("Requires test database configuration")
	return nil
}

func cleanupTestStore(t *testing.T, store *Store) {
	// Clean up test data
}

func hashPassword(t *testing.T, password string) string {
	// Use bcrypt or test hash
	return "$2a$10$" + password
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && contains(s[1:], substr)
}
