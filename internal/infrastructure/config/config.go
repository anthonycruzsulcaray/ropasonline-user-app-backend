package config

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

type Config struct {
	HTTPAddress string
	MySQLDSN    string
	JWTSecret   string
	JWTDuration time.Duration
}

func Load() (Config, error) {
	host, err := required("DB_HOST")
	if err != nil {
		return Config{}, err
	}
	port, err := required("DB_PORT")
	if err != nil {
		return Config{}, err
	}
	database, err := required("DB_NAME")
	if err != nil {
		return Config{}, err
	}
	username, err := required("DB_USER")
	if err != nil {
		return Config{}, err
	}
	password, err := required("DB_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	secret, err := required("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	if len(secret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	expirationSecondsText, err := required("JWT_EXPIRATION_SECONDS")
	if err != nil {
		return Config{}, err
	}
	expirationSeconds, err := strconv.ParseInt(expirationSecondsText, 10, 64)
	if err != nil || expirationSeconds <= 0 || expirationSeconds > math.MaxInt64/int64(time.Second) {
		return Config{}, errors.New("JWT_EXPIRATION_SECONDS must be a positive integer within the supported range")
	}

	dbConfig := mysqldriver.NewConfig()
	dbConfig.User = username
	dbConfig.Passwd = password
	dbConfig.Net = "tcp"
	dbConfig.Addr = net.JoinHostPort(host, port)
	dbConfig.DBName = database
	dbConfig.ParseTime = true
	dbConfig.Loc = time.UTC
	dbConfig.Params = map[string]string{"charset": "utf8mb4"}

	return Config{
		HTTPAddress: envOrDefault("HTTP_ADDRESS", ":8080"),
		MySQLDSN:    dbConfig.FormatDSN(),
		JWTSecret:   secret,
		JWTDuration: time.Duration(expirationSeconds) * time.Second,
	}, nil
}

func required(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return "", fmt.Errorf("required environment variable %s is missing", key)
	}
	return value, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
