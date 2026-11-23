package test

import (
	"context"
	"errors"
	"strings"
	"testing"

	applicationauth "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application/auth"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
)

type testUserRepository struct {
	users  map[uint64]user.User
	nextID uint64
}

func newTestUserRepository() *testUserRepository {
	return &testUserRepository{users: make(map[uint64]user.User), nextID: 1}
}

func (r *testUserRepository) Create(_ context.Context, newUser user.User) (user.User, error) {
	for _, existing := range r.users {
		if existing.Email == newUser.Email {
			return user.User{}, applicationauth.ErrEmailAlreadyExists
		}
	}
	newUser.ID = r.nextID
	r.nextID++
	r.users[newUser.ID] = newUser
	return newUser, nil
}

func (r *testUserRepository) FindByEmail(_ context.Context, email string) (user.User, error) {
	for _, existing := range r.users {
		if existing.Email == email {
			return existing, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (r *testUserRepository) FindByUUID(_ context.Context, publicUUID string) (user.User, error) {
	for _, found := range r.users {
		if found.UUID == publicUUID {
			return found, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (r *testUserRepository) Update(_ context.Context, updated user.User) (user.User, error) {
	if _, exists := r.users[updated.ID]; !exists {
		return user.User{}, user.ErrNotFound
	}
	for id, existing := range r.users {
		if id != updated.ID && existing.Email == updated.Email {
			return user.User{}, applicationauth.ErrEmailAlreadyExists
		}
	}
	r.users[updated.ID] = updated
	return updated, nil
}

func (r *testUserRepository) Delete(_ context.Context, publicUUID string) error {
	for id, found := range r.users {
		if found.UUID == publicUUID {
			delete(r.users, id)
			return nil
		}
	}
	return user.ErrNotFound
}

type testPasswordManager struct{}

func (testPasswordManager) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (testPasswordManager) Compare(password, hash string) error {
	if hash != "hash:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

type testTokenManager struct{}

func (testTokenManager) Issue(userUUID string) (string, error) {
	return "token:" + userUUID, nil
}

func (testTokenManager) Parse(token string) (string, error) {
	const prefix = "token:"
	if !strings.HasPrefix(token, prefix) {
		return "", errors.New("invalid test token")
	}
	return strings.TrimPrefix(token, prefix), nil
}

func TestRegisterAndLogin(t *testing.T) {
	repository := newTestUserRepository()
	service := applicationauth.NewService(repository, testPasswordManager{}, testTokenManager{})

	registered, err := service.Register(context.Background(), "Ana", "Cruz", "ANA@example.com", "password123")
	if err != nil {
		t.Fatalf("Register returned an error: %v", err)
	}
	if registered.Name != "Ana" || registered.LastName != "Cruz" || registered.Email != "ana@example.com" || registered.PasswordHash != "hash:password123" {
		t.Fatalf("unexpected registered user: %#v", registered)
	}

	loginResult, err := service.Login(context.Background(), "ana@example.com", "password123")
	if err != nil {
		t.Fatalf("Login returned an error: %v", err)
	}
	if loginResult.Token != "token:"+registered.UUID {
		t.Fatalf("unexpected token: %q", loginResult.Token)
	}
	if loginResult.User.UUID != registered.UUID {
		t.Fatalf("expected login user UUID %q, got %q", registered.UUID, loginResult.User.UUID)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	service := applicationauth.NewService(newTestUserRepository(), testPasswordManager{}, testTokenManager{})
	_, err := service.Register(context.Background(), "Ana", "Cruz", "ana@example.com", "short")
	if !errors.Is(err, applicationauth.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLoginDoesNotRevealUnknownEmail(t *testing.T) {
	service := applicationauth.NewService(newTestUserRepository(), testPasswordManager{}, testTokenManager{})
	_, err := service.Login(context.Background(), "missing@example.com", "password123")
	if !errors.Is(err, applicationauth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateRejectsMissingUser(t *testing.T) {
	service := applicationauth.NewService(newTestUserRepository(), testPasswordManager{}, testTokenManager{})
	_, err := service.Authenticate(context.Background(), "token:550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, applicationauth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
