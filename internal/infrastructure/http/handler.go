package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application"
)

type authUseCase interface {
	Register(ctx context.Context, email, password string) (bool, error)
	Confirm(ctx context.Context, email, code string) error
	ResendConfirmation(ctx context.Context, email string) (application.CodeDelivery, error)
	Login(ctx context.Context, email, password string) (application.Tokens, error)
}

type handler struct {
	auth authUseCase
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type confirmRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RegisterRoutes(router gin.IRouter, auth authUseCase) {
	h := &handler{auth: auth}
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	group := router.Group("/api/auth")
	group.POST("/register", h.register)
	group.POST("/confirm", h.confirm)
	group.POST("/resend-confirmation", h.resendConfirmation)
	group.POST("/login", h.login)
}

func (h *handler) register(c *gin.Context) {
	var request credentialsRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.Email = strings.TrimSpace(request.Email)
	if !validEmail(request.Email) || request.Password == "" {
		writeError(c, http.StatusBadRequest, "invalid_request", "Se requiere un correo válido y una contraseña.")
		return
	}

	confirmed, err := h.auth.Register(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":               "Registro completado.",
		"confirmation_required": !confirmed,
	})
}

func (h *handler) confirm(c *gin.Context) {
	var request confirmRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.Email = strings.TrimSpace(request.Email)
	request.Code = strings.TrimSpace(request.Code)
	if !validEmail(request.Email) || request.Code == "" {
		writeError(c, http.StatusBadRequest, "invalid_request", "Se requiere un correo válido y el código de confirmación.")
		return
	}
	for _, digit := range request.Code {
		if digit < '0' || digit > '9' {
			writeError(c, http.StatusBadRequest, "invalid_request", "El código de confirmación debe contener solo dígitos.")
			return
		}
	}

	if err := h.auth.Confirm(c.Request.Context(), request.Email, request.Code); err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cuenta confirmada."})
}

func (h *handler) resendConfirmation(c *gin.Context) {
	var request emailRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.Email = strings.TrimSpace(request.Email)
	if !validEmail(request.Email) {
		writeError(c, http.StatusBadRequest, "invalid_request", "Se requiere un correo válido.")
		return
	}

	delivery, err := h.auth.ResendConfirmation(c.Request.Context(), request.Email)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response := gin.H{"message": "Código de confirmación reenviado."}
	if delivery.DeliveryMedium != "" {
		response["delivery_medium"] = delivery.DeliveryMedium
	}
	if delivery.Destination != "" {
		response["destination"] = delivery.Destination
	}
	c.JSON(http.StatusOK, response)
}

func (h *handler) login(c *gin.Context) {
	var request credentialsRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.Email = strings.TrimSpace(request.Email)
	if !validEmail(request.Email) || request.Password == "" {
		writeError(c, http.StatusBadRequest, "invalid_request", "Se requiere un correo válido y una contraseña.")
		return
	}

	tokens, err := h.auth.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, tokens)
}

func decodeJSON(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "El cuerpo JSON no es válido.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "invalid_request", "El cuerpo debe contener un solo objeto JSON.")
		return false
	}
	return true
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrEmailAlreadyExists):
		writeError(c, http.StatusConflict, "email_already_exists", "Ya existe una cuenta con ese correo.")
	case errors.Is(err, application.ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "Correo o contraseña incorrectos.")
	case errors.Is(err, application.ErrUserNotConfirmed):
		writeError(c, http.StatusForbidden, "user_not_confirmed", "Confirma tu cuenta antes de iniciar sesión.")
	case errors.Is(err, application.ErrInvalidConfirmationCode):
		writeError(c, http.StatusBadRequest, "invalid_confirmation_code", "El código de confirmación no es válido.")
	case errors.Is(err, application.ErrExpiredConfirmationCode):
		writeError(c, http.StatusBadRequest, "expired_confirmation_code", "El código de confirmación expiró.")
	case errors.Is(err, application.ErrWeakPassword):
		writeError(c, http.StatusBadRequest, "weak_password", "La contraseña no cumple los requisitos.")
	case errors.Is(err, application.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_request", "Los datos enviados no son válidos.")
	case errors.Is(err, application.ErrRateLimited):
		writeError(c, http.StatusTooManyRequests, "too_many_requests", "Demasiados intentos. Inténtalo más tarde.")
	case errors.Is(err, application.ErrUserNotFound):
		writeError(c, http.StatusNotFound, "user_not_found", "No existe una cuenta con ese correo.")
	case errors.Is(err, application.ErrUserAlreadyConfirmed):
		writeError(c, http.StatusConflict, "user_already_confirmed", "La cuenta ya está confirmada.")
	case errors.Is(err, application.ErrCodeDeliveryFailure):
		writeError(c, http.StatusBadGateway, "code_delivery_failure", "Cognito no pudo enviar el código. Inténtalo más tarde.")
	default:
		log.Printf("auth provider error: %v", err)
		writeError(c, http.StatusBadGateway, "auth_provider_error", "No fue posible completar la solicitud de autenticación.")
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorResponse{
		Error: errorDetail{Code: code, Message: message},
	})
}
