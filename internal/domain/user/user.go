package user

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("user not found")
	ErrInvalidUser = errors.New("invalid user")
)

type User struct {
	ID           uint64
	UUID         string
	Name         string
	LastName     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

func New(name, lastName, email, passwordHash string) (User, error) {
	name = strings.TrimSpace(name)
	lastName = strings.TrimSpace(lastName)
	email = NormalizeEmail(email)
	if name == "" || len(name) > 100 {
		return User{}, errors.Join(ErrInvalidUser, errors.New("name must contain between 1 and 100 characters"))
	}
	if lastName == "" || len(lastName) > 100 {
		return User{}, errors.Join(ErrInvalidUser, errors.New("last name must contain between 1 and 100 characters"))
	}
	if !isValidEmail(email) {
		return User{}, errors.Join(ErrInvalidUser, errors.New("email is invalid"))
	}
	if passwordHash == "" {
		return User{}, errors.Join(ErrInvalidUser, errors.New("password hash is required"))
	}

	now := time.Now().UTC()
	return User{
		UUID:         uuid.NewString(),
		Name:         name,
		LastName:     lastName,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}, nil
}

func (u *User) UpdateProfile(name, lastName, email string) error {
	name = strings.TrimSpace(name)
	lastName = strings.TrimSpace(lastName)
	email = NormalizeEmail(email)
	if name == "" || len(name) > 100 {
		return errors.Join(ErrInvalidUser, errors.New("name must contain between 1 and 100 characters"))
	}
	if lastName == "" || len(lastName) > 100 {
		return errors.Join(ErrInvalidUser, errors.New("last name must contain between 1 and 100 characters"))
	}
	if !isValidEmail(email) {
		return errors.Join(ErrInvalidUser, errors.New("email is invalid"))
	}

	u.Name = name
	u.LastName = lastName
	u.Email = email
	updatedAt := time.Now().UTC()
	u.UpdatedAt = &updatedAt
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}
