package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application/auth"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
	httpapi "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/transport/http"
	"github.com/gin-gonic/gin"
)

type testAuthService struct{}

const testUserUUID = "550e8400-e29b-41d4-a716-446655440000"

func (testAuthService) Register(_ context.Context, name, lastName, email, _ string) (user.User, error) {
	return user.User{
		ID: 1, UUID: testUserUUID, Name: name, LastName: lastName, Email: email, PasswordHash: "must-not-be-returned",
		CreatedAt: time.Now(),
	}, nil
}

func (testAuthService) Login(context.Context, string, string) (applicationauth.LoginResult, error) {
	return applicationauth.LoginResult{
		Token: "test-token",
		User:  user.User{UUID: testUserUUID, Name: "Ana", LastName: "Cruz", Email: "ana@example.com"},
	}, nil
}

func (testAuthService) Profile(context.Context, string) (user.User, error) {
	return user.User{ID: 7, UUID: testUserUUID, Name: "Ana", LastName: "Cruz", Email: "ana@example.com"}, nil
}

func (testAuthService) UpdateProfile(context.Context, string, string, string, string) (user.User, error) {
	return user.User{UUID: testUserUUID, Name: "Ana", LastName: "Cruz", Email: "ana@example.com"}, nil
}

func (testAuthService) DeleteProfile(context.Context, string) error {
	return nil
}

func (testAuthService) Authenticate(_ context.Context, token string) (string, error) {
	if token != "valid-token" {
		return "", user.ErrNotFound
	}
	return testUserUUID, nil
}

func TestRegisterDoesNotExposePasswordHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	httpapi.RegisterRoutes(router, testAuthService{}, 5*time.Minute)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(
		`{"name":"Ana","last_name":"Cruz","email":"ana@example.com","password":"password123"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := body["password_hash"]; exists {
		t.Fatal("registration response exposed the password hash")
	}
	if body["uuid"] != testUserUUID {
		t.Fatalf("expected response UUID %q, got %v", testUserUUID, body["uuid"])
	}
	if body["last_name"] != "Cruz" {
		t.Fatalf("expected last_name Cruz, got %v", body["last_name"])
	}
	if body["updated_at"] != nil {
		t.Fatalf("expected updated_at null for a new user, got %v", body["updated_at"])
	}
	if _, exists := body["id"]; exists {
		t.Fatal("registration response exposed the internal numeric ID")
	}
}

func TestProfileRequiresBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	httpapi.RegisterRoutes(router, testAuthService{}, 5*time.Minute)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestLoginReturnsConfiguredTokenExpiration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	httpapi.RegisterRoutes(router, testAuthService{}, 5*time.Minute)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(
		`{"email":"ana@example.com","password":"password123"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var body struct {
		ExpiresIn int64 `json:"expires_in"`
		User      struct {
			UUID     string `json:"uuid"`
			LastName string `json:"last_name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ExpiresIn != 300 {
		t.Fatalf("expected expires_in 300, got %d", body.ExpiresIn)
	}
	if body.User.UUID != testUserUUID {
		t.Fatalf("expected login user UUID %q, got %q", testUserUUID, body.User.UUID)
	}
	if body.User.LastName != "Cruz" {
		t.Fatalf("expected login user last_name Cruz, got %q", body.User.LastName)
	}
}
