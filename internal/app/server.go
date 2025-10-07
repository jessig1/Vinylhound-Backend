package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrUserExists signals the username is already taken.
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials indicates a login failure.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUnauthorized indicates an invalid or missing session.
	ErrUnauthorized = errors.New("unauthorized")

	dummyPasswordHash = []byte("$2a$10$CwTycUXWue0Thq9StjUM0uJ8n4VWeNseyX2fA9DE.D7su7J6iYGTC")
)

// Store provides persistence backed by Postgres.
type Store struct {
	db *sql.DB
}

// NewStore sets up a Store using the provided database handle.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateUser registers a new user with optional starter content.
func (s *Store) CreateUser(username, password string, content []string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var userID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`, username, hash).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserExists
		}
		return fmt.Errorf("insert user: %w", err)
	}

	for i, entry := range content {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_content (user_id, position, entry)
			VALUES ($1, $2, $3)
		`, userID, i, entry); err != nil {
			return fmt.Errorf("insert content: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	tx = nil

	return nil
}

// Authenticate validates credentials and returns a session token.
func (s *Store) Authenticate(username, password string) (string, error) {
	ctx := context.Background()

	var (
		userID int64
		hash   []byte
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash
		FROM users
		WHERE username = $1
	`, username).Scan(&userID, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("lookup user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id)
		VALUES ($1, $2)
	`, token, userID); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}

	return token, nil
}

// ContentByToken returns user-specific content for a valid token.
func (s *Store) ContentByToken(token string) ([]string, error) {
	ctx := context.Background()

	userID, err := s.userIDForToken(ctx, token)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT entry
		FROM user_content
		WHERE user_id = $1
		ORDER BY position ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("select content: %w", err)
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

// UpdateContentByToken replaces the content owned by the authenticated user.
func (s *Store) UpdateContentByToken(token string, content []string) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	userID, err := s.userIDForTokenTx(ctx, tx, token)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM user_content
		WHERE user_id = $1
	`, userID); err != nil {
		return fmt.Errorf("delete content: %w", err)
	}

	for i, entry := range content {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_content (user_id, position, entry)
			VALUES ($1, $2, $3)
		`, userID, i, entry); err != nil {
			return fmt.Errorf("insert content: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	tx = nil

	return nil
}

func (s *Store) userIDForToken(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUnauthorized
		}
		return 0, fmt.Errorf("lookup session: %w", err)
	}
	return userID, nil
}

func (s *Store) userIDForTokenTx(ctx context.Context, tx *sql.Tx, token string) (int64, error) {
	var userID int64
	err := tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM sessions
		WHERE token = $1
	`, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUnauthorized
		}
		return 0, fmt.Errorf("lookup session: %w", err)
	}
	return userID, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Server wires HTTP handlers to a Store.
type Server struct {
	store *Store
}

// NewServer configures a Server with the given Store.
func NewServer(store *Store) *Server {
	return &Server{store: store}
}

// Routes exposes the HTTP handlers for account and content management.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/signup", s.handleSignup)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/me/content", s.handleContent)
	return mux
}

type signupRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Content  []string `json:"content"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	if err := s.store.CreateUser(req.Username, req.Password, req.Content); err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "username already taken"})
		default:
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	token, err := s.store.Authenticate(req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, ErrInvalidCredentials) {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		content, err := s.store.ContentByToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			if !errors.Is(err, ErrUnauthorized) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Content []string `json:"content"`
		}{Content: content})
	case http.MethodPut:
		var body struct {
			Content []string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
			return
		}
		if err := s.store.UpdateContentByToken(token, body.Content); err != nil {
			status := http.StatusUnauthorized
			if !errors.Is(err, ErrUnauthorized) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
