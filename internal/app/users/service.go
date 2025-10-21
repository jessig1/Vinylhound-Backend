package users

import "context"

// Store describes the persistence operations required by the user service.
type Store interface {
	CreateUser(username, password string, content []string) error
	Authenticate(username, password string) (string, error)
	ContentByToken(token string) ([]string, error)
	UpdateContentByToken(token string, content []string) error
}

// Service exposes user-related workflows in an extensible manner.
type Service interface {
	Signup(ctx context.Context, username, password string, content []string) error
	Authenticate(ctx context.Context, username, password string) (string, error)
	Content(ctx context.Context, token string) ([]string, error)
	UpdateContent(ctx context.Context, token string, content []string) error
}

type service struct {
	store Store
}

// New wires a Service backed by the provided Store.
func New(store Store) Service {
	return &service{store: store}
}

func (s *service) Signup(ctx context.Context, username, password string, content []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.CreateUser(username, password, content)
}

func (s *service) Authenticate(ctx context.Context, username, password string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.store.Authenticate(username, password)
}

func (s *service) Content(ctx context.Context, token string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.store.ContentByToken(token)
}

func (s *service) UpdateContent(ctx context.Context, token string, content []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.store.UpdateContentByToken(token, content)
}
