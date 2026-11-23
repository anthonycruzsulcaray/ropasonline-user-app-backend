package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must contain at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("JWT expiration must be positive")
	}
	return &JWTManager{secret: []byte(secret), ttl: ttl}, nil
}

func (m *JWTManager) Issue(userUUID string) (string, error) {
	parsedUUID, err := uuid.Parse(userUUID)
	if err != nil {
		return "", errors.New("user UUID is invalid")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   parsedUUID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) Parse(rawToken string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT signing method")
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	userUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return "", errors.New("invalid token subject")
	}
	return userUUID.String(), nil
}
