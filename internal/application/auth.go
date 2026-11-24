package application

import (
	"context"
	"errors"
)

var (
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUserNotConfirmed        = errors.New("user is not confirmed")
	ErrInvalidConfirmationCode = errors.New("invalid confirmation code")
	ErrExpiredConfirmationCode = errors.New("confirmation code expired")
	ErrWeakPassword            = errors.New("password does not meet requirements")
	ErrInvalidInput            = errors.New("invalid input")
	ErrRateLimited             = errors.New("too many requests")
	ErrUserNotFound            = errors.New("user not found")
	ErrUserAlreadyConfirmed    = errors.New("user is already confirmed")
	ErrCodeDeliveryFailure     = errors.New("confirmation code delivery failed")
)

type CognitoPort interface {
	SignUp(ctx context.Context, email, password string) (confirmed bool, err error)
	ConfirmSignUp(ctx context.Context, email, code string) error
	ResendConfirmationCode(ctx context.Context, email string) (CodeDelivery, error)
	Login(ctx context.Context, email, password string) (Tokens, error)
}

type CodeDelivery struct {
	DeliveryMedium string
	Destination    string
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int32  `json:"expires_in"`
}

type AuthService struct {
	cognito CognitoPort
}

func NewAuthService(cognito CognitoPort) *AuthService {
	return &AuthService{cognito: cognito}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (bool, error) {
	return s.cognito.SignUp(ctx, email, password)
}

func (s *AuthService) Confirm(ctx context.Context, email, code string) error {
	return s.cognito.ConfirmSignUp(ctx, email, code)
}

func (s *AuthService) ResendConfirmation(ctx context.Context, email string) (CodeDelivery, error) {
	return s.cognito.ResendConfirmationCode(ctx, email)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (Tokens, error) {
	return s.cognito.Login(ctx, email, password)
}
