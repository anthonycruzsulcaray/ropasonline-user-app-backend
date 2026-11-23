package test

import (
	"errors"
	"testing"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
	"github.com/google/uuid"
)

func TestNewNormalizesNameLastNameAndEmail(t *testing.T) {
	found, err := user.New("  Ana  ", " Cruz ", " ANA@example.com ", "hashed-password")
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}
	if found.Name != "Ana" || found.LastName != "Cruz" || found.Email != "ana@example.com" {
		t.Fatalf("unexpected normalized user: %#v", found)
	}
	if found.UpdatedAt != nil {
		t.Fatalf("expected UpdatedAt to be nil for a new user, got %v", found.UpdatedAt)
	}
	if _, err := uuid.Parse(found.UUID); err != nil {
		t.Fatalf("constructor returned an invalid UUID: %v", err)
	}
}

func TestNewRejectsInvalidEmail(t *testing.T) {
	_, err := user.New("Ana", "Cruz", "not-an-email", "hashed-password")
	if !errors.Is(err, user.ErrInvalidUser) {
		t.Fatalf("expected ErrInvalidUser, got %v", err)
	}
}
