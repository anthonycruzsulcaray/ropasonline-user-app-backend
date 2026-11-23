package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
)

type UserRepository interface {
	Create(context.Context, user.User) (user.User, error)
	FindByEmail(context.Context, string) (user.User, error)
	FindByUUID(context.Context, string) (user.User, error)
	Update(context.Context, user.User) (user.User, error)
	Delete(context.Context, string) error
}

type PasswordManager interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type TokenManager interface {
	Issue(string) (string, error)
	Parse(string) (string, error)
}

type LoginResult struct {
	Token string
	User  user.User
}

type Service struct {
	users     UserRepository
	passwords PasswordManager
	tokens    TokenManager
}

func NewService(users UserRepository, passwords PasswordManager, tokens TokenManager) *Service {
	return &Service{users: users, passwords: passwords, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, name, lastName, email, password string) (user.User, error) {
	if len(password) < 8 || len(password) > 72 {
		return user.User{}, errors.Join(ErrInvalidInput, errors.New("password must contain between 8 and 72 bytes"))
	}

	hash, err := s.passwords.Hash(password)
	if err != nil {
		return user.User{}, err
	}
	newUser, err := user.New(name, lastName, email, hash)
	if err != nil {
		return user.User{}, err
	}

	return s.users.Create(ctx, newUser)
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	storedUser, err := s.users.FindByEmail(ctx, user.NormalizeEmail(email))
	if errors.Is(err, user.ErrNotFound) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user for login: %w", err)
	}
	if s.passwords.Compare(password, storedUser.PasswordHash) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	token, err := s.tokens.Issue(storedUser.UUID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue access token: %w", err)
	}
	return LoginResult{Token: token, User: storedUser}, nil
}

func (s *Service) Profile(ctx context.Context, publicUUID string) (user.User, error) {
	return s.users.FindByUUID(ctx, publicUUID)
}

func (s *Service) UpdateProfile(ctx context.Context, publicUUID, name, lastName, email string) (user.User, error) {
	storedUser, err := s.users.FindByUUID(ctx, publicUUID)
	if err != nil {
		return user.User{}, err
	}
	if err := storedUser.UpdateProfile(name, lastName, email); err != nil {
		return user.User{}, err
	}
	return s.users.Update(ctx, storedUser)
}

func (s *Service) DeleteProfile(ctx context.Context, publicUUID string) error {
	return s.users.Delete(ctx, publicUUID)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (string, error) {
	publicUUID, err := s.tokens.Parse(strings.TrimSpace(rawToken))
	if err != nil {
		return "", ErrUnauthorized
	}
	if _, err := s.users.FindByUUID(ctx, publicUUID); errors.Is(err, user.ErrNotFound) {
		return "", ErrUnauthorized
	} else if err != nil {
		return "", fmt.Errorf("verify token user: %w", err)
	}
	return publicUUID, nil
}
