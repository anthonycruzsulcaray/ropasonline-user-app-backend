package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/config"
)

func TestLoadDotEnvFromParentDirectory(t *testing.T) {
	root := t.TempDir()
	startDir := filepath.Join(root, "cmd", "api")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("create nested working directory: %v", err)
	}
	dotenv := filepath.Join(root, ".env")
	if err := os.WriteFile(dotenv, []byte("AWS_REGION=us-west-1\nCOGNITO_CLIENT_ID=from-file\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("AWS_REGION", "")
	t.Setenv("COGNITO_CLIENT_ID", "from-environment")
	if err := config.LoadDotEnvFrom(startDir); err != nil {
		t.Fatalf("load .env: %v", err)
	}

	if got := os.Getenv("AWS_REGION"); got != "us-west-1" {
		t.Errorf("AWS_REGION = %q, want %q", got, "us-west-1")
	}
	if got := os.Getenv("COGNITO_CLIENT_ID"); got != "from-environment" {
		t.Errorf("COGNITO_CLIENT_ID = %q, want existing environment value to take priority", got)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load application config: %v", err)
	}
	if cfg.AWSRegion != "us-west-1" {
		t.Errorf("config AWS region = %q, want %q", cfg.AWSRegion, "us-west-1")
	}
}
