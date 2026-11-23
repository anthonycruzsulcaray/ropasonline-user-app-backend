package test

import (
	"testing"
	"time"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/security"
)

func TestBcryptPasswordManager(t *testing.T) {
	manager := security.BcryptPasswordManager{}
	hash, err := manager.Hash("password123")
	if err != nil {
		t.Fatalf("Hash returned an error: %v", err)
	}
	if hash == "password123" {
		t.Fatal("password was not hashed")
	}
	if err := manager.Compare("password123", hash); err != nil {
		t.Fatalf("Compare rejected the correct password: %v", err)
	}
	if err := manager.Compare("incorrect", hash); err == nil {
		t.Fatal("Compare accepted an incorrect password")
	}
}

func TestJWTManagerRoundTrip(t *testing.T) {
	manager, err := security.NewJWTManager("0123456789abcdef0123456789abcdef", time.Minute)
	if err != nil {
		t.Fatalf("NewJWTManager returned an error: %v", err)
	}
	const userUUID = "550e8400-e29b-41d4-a716-446655440000"
	token, err := manager.Issue(userUUID)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}
	parsedUUID, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if parsedUUID != userUUID {
		t.Fatalf("expected UUID %s, got %s", userUUID, parsedUUID)
	}
}
