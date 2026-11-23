package test

import (
	"testing"
	"time"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/config"
)

func setConfigEnvironment(t *testing.T, jwtExpirationSeconds string) {
	t.Helper()
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_NAME", "ropas_online_auth")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_EXPIRATION_SECONDS", jwtExpirationSeconds)
}

func TestLoadReadsJWTExpirationSeconds(t *testing.T) {
	setConfigEnvironment(t, "300")

	settings, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if settings.JWTDuration != 300*time.Second {
		t.Fatalf("expected JWT duration %s, got %s", 300*time.Second, settings.JWTDuration)
	}
}

func TestLoadRejectsInvalidJWTExpirationSeconds(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc", "999999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			setConfigEnvironment(t, value)
			if _, err := config.Load(); err == nil {
				t.Fatal("expected invalid JWT expiration to be rejected")
			}
		})
	}
}
