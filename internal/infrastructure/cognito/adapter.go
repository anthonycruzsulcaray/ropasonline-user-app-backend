package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/smithy-go"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application"
)

type Adapter struct {
	client       *cognitoidentityprovider.Client
	clientID     string
	clientSecret string
}

func New(cfg aws.Config, clientID, clientSecret string) *Adapter {
	return &Adapter{
		client:       cognitoidentityprovider.NewFromConfig(cfg),
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (a *Adapter) SignUp(ctx context.Context, email, password string) (bool, error) {
	input := &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(a.clientID),
		Password: aws.String(password),
		Username: aws.String(email),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
	}
	if a.clientSecret != "" {
		input.SecretHash = aws.String(a.secretHash(email))
	}

	output, err := a.client.SignUp(ctx, input)
	if err != nil {
		return false, mapCognitoError("sign up", err)
	}
	return output.UserConfirmed, nil
}

func (a *Adapter) ConfirmSignUp(ctx context.Context, email, code string) error {
	input := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(a.clientID),
		ConfirmationCode: aws.String(code),
		Username:         aws.String(email),
	}
	if a.clientSecret != "" {
		input.SecretHash = aws.String(a.secretHash(email))
	}

	if _, err := a.client.ConfirmSignUp(ctx, input); err != nil {
		return mapCognitoError("confirm sign up", err)
	}
	return nil
}

func (a *Adapter) ResendConfirmationCode(ctx context.Context, email string) (application.CodeDelivery, error) {
	input := &cognitoidentityprovider.ResendConfirmationCodeInput{
		ClientId: aws.String(a.clientID),
		Username: aws.String(email),
	}
	if a.clientSecret != "" {
		input.SecretHash = aws.String(a.secretHash(email))
	}

	output, err := a.client.ResendConfirmationCode(ctx, input)
	if err != nil {
		return application.CodeDelivery{}, mapCognitoError("resend confirmation", err)
	}

	delivery := application.CodeDelivery{}
	if output.CodeDeliveryDetails != nil {
		delivery.DeliveryMedium = string(output.CodeDeliveryDetails.DeliveryMedium)
		if output.CodeDeliveryDetails.Destination != nil {
			delivery.Destination = *output.CodeDeliveryDetails.Destination
		}
	}
	return delivery, nil
}

func (a *Adapter) Login(ctx context.Context, email, password string) (application.Tokens, error) {
	authParameters := map[string]string{
		"USERNAME": email,
		"PASSWORD": password,
	}
	if a.clientSecret != "" {
		authParameters["SECRET_HASH"] = a.secretHash(email)
	}

	output, err := a.client.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
		ClientId:       aws.String(a.clientID),
		AuthParameters: authParameters,
	})
	if err != nil {
		return application.Tokens{}, mapCognitoError("login", err)
	}
	if output.AuthenticationResult == nil || output.AuthenticationResult.AccessToken == nil {
		return application.Tokens{}, fmt.Errorf("login: Cognito did not return an access token")
	}

	result := output.AuthenticationResult
	tokens := application.Tokens{
		AccessToken: aws.ToString(result.AccessToken),
		TokenType:   aws.ToString(result.TokenType),
	}
	if result.IdToken != nil {
		tokens.IDToken = *result.IdToken
	}
	if result.RefreshToken != nil {
		tokens.RefreshToken = *result.RefreshToken
	}
	tokens.ExpiresIn = result.ExpiresIn
	return tokens, nil
}

func (a *Adapter) secretHash(username string) string {
	mac := hmac.New(sha256.New, []byte(a.clientSecret))
	_, _ = mac.Write([]byte(username + a.clientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func mapCognitoError(operation string, err error) error {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("cognito %s: %w", operation, err)
	}

	var mapped error
	switch apiErr.ErrorCode() {
	case "UsernameExistsException":
		mapped = application.ErrEmailAlreadyExists
	case "NotAuthorizedException":
		if operation == "resend confirmation" {
			mapped = application.ErrUserAlreadyConfirmed
		} else {
			mapped = application.ErrInvalidCredentials
		}
	case "UserNotFoundException":
		if operation == "resend confirmation" {
			mapped = application.ErrUserNotFound
		} else {
			mapped = application.ErrInvalidCredentials
		}
	case "CodeDeliveryFailureException":
		mapped = application.ErrCodeDeliveryFailure
	case "UserNotConfirmedException":
		mapped = application.ErrUserNotConfirmed
	case "CodeMismatchException":
		mapped = application.ErrInvalidConfirmationCode
	case "ExpiredCodeException":
		mapped = application.ErrExpiredConfirmationCode
	case "InvalidPasswordException":
		mapped = application.ErrWeakPassword
	case "InvalidParameterException":
		mapped = application.ErrInvalidInput
	case "TooManyRequestsException", "LimitExceededException":
		mapped = application.ErrRateLimited
	default:
		return fmt.Errorf("cognito %s: %w", operation, err)
	}
	return fmt.Errorf("cognito %s: %w", operation, mapped)
}
