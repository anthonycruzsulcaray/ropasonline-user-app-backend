package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AWSRegion           string
	CognitoClientID     string
	CognitoClientSecret string
	HTTPAddr            string
}

func LoadDotEnvFrom(startDir string) error {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return fmt.Errorf("resolve .env search directory: %w", err)
	}

	for {
		path := filepath.Join(dir, ".env")
		values, err := godotenv.Read(path)
		if err == nil {
			for key, value := range values {
				if current, exists := os.LookupEnv(key); exists && current != "" {
					continue
				}
				if err := os.Setenv(key, value); err != nil {
					return fmt.Errorf("set environment variable %s from .env: %w", key, err)
				}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read .env at %s: %w", path, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func Load() (Config, error) {
	cfg := Config{
		AWSRegion:           strings.TrimSpace(os.Getenv("AWS_REGION")),
		CognitoClientID:     strings.TrimSpace(os.Getenv("COGNITO_CLIENT_ID")),
		CognitoClientSecret: strings.TrimSpace(os.Getenv("COGNITO_CLIENT_SECRET")),
		HTTPAddr:            strings.TrimSpace(os.Getenv("HTTP_ADDR")),
	}

	if cfg.AWSRegion == "" {
		return Config{}, fmt.Errorf("AWS_REGION is required")
	}
	if cfg.CognitoClientID == "" {
		return Config{}, fmt.Errorf("COGNITO_CLIENT_ID is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	return cfg, nil
}
