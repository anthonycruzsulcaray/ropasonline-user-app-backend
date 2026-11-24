package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application"
	httpapi "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/http"
)

type fakeAuth struct {
	registerCalls  int
	confirmCalls   int
	resendCalls    int
	loginCalls     int
	registerOK     bool
	resendDelivery application.CodeDelivery
	loginTokens    application.Tokens
	err            error
}

func (f *fakeAuth) Register(_ context.Context, _, _ string) (bool, error) {
	f.registerCalls++
	return f.registerOK, f.err
}

func (f *fakeAuth) Confirm(_ context.Context, _, _ string) error {
	f.confirmCalls++
	return f.err
}

func (f *fakeAuth) ResendConfirmation(_ context.Context, _ string) (application.CodeDelivery, error) {
	f.resendCalls++
	return f.resendDelivery, f.err
}

func (f *fakeAuth) Login(_ context.Context, _, _ string) (application.Tokens, error) {
	f.loginCalls++
	return f.loginTokens, f.err
}

func newRouter(auth *fakeAuth) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	httpapi.RegisterRoutes(router, auth)
	return router
}

func TestRegisterRejectsInvalidEmailWithoutCallingProvider(t *testing.T) {
	auth := &fakeAuth{}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register",
		bytes.NewBufferString(`{"email":"not-an-email","password":"StrongPass123!"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if auth.registerCalls != 0 {
		t.Fatalf("provider calls = %d, want 0", auth.registerCalls)
	}
}

func TestRegisterRequestsConfirmationWhenCognitoRequiresIt(t *testing.T) {
	auth := &fakeAuth{registerOK: false}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register",
		bytes.NewBufferString(`{"email":"user@example.com","password":"StrongPass123!"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["confirmation_required"] != true {
		t.Fatalf("confirmation_required = %v, want true", body["confirmation_required"])
	}
	if auth.registerCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", auth.registerCalls)
	}
}

func TestConfirmAcceptsCognitoCode(t *testing.T) {
	auth := &fakeAuth{}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/confirm",
		bytes.NewBufferString(`{"email":"user@example.com","code":"123456"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if auth.confirmCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", auth.confirmCalls)
	}
}

func TestResendConfirmationReturnsDeliveryDetails(t *testing.T) {
	auth := &fakeAuth{resendDelivery: application.CodeDelivery{
		DeliveryMedium: "EMAIL",
		Destination:    "u***@example.com",
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/resend-confirmation",
		bytes.NewBufferString(`{"email":"user@example.com"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["delivery_medium"] != "EMAIL" || body["destination"] != "u***@example.com" {
		t.Fatalf("unexpected delivery details: %v", body)
	}
	if auth.resendCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", auth.resendCalls)
	}
}

func TestLoginReturnsCognitoTokens(t *testing.T) {
	auth := &fakeAuth{loginTokens: application.Tokens{
		AccessToken: "access",
		IDToken:     "identity",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		bytes.NewBufferString(`{"email":"user@example.com","password":"StrongPass123!"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["access_token"] != "access" || body["id_token"] != "identity" {
		t.Fatalf("unexpected token response: %v", body)
	}
	if auth.loginCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", auth.loginCalls)
	}
}

func TestConfirmRejectsNonDigitCode(t *testing.T) {
	auth := &fakeAuth{}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/confirm",
		bytes.NewBufferString(`{"email":"user@example.com","code":"12ab56"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if auth.confirmCalls != 0 {
		t.Fatalf("provider calls = %d, want 0", auth.confirmCalls)
	}
}

func TestLoginDoesNotExposeProviderError(t *testing.T) {
	auth := &fakeAuth{err: errors.New("sensitive provider details")}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		bytes.NewBufferString(`{"email":"user@example.com","password":"StrongPass123!"}`))
	response := httptest.NewRecorder()

	newRouter(auth).ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
	}
	if bytes.Contains(response.Body.Bytes(), []byte("sensitive provider details")) {
		t.Fatalf("provider error leaked to client: %s", response.Body.String())
	}
}
