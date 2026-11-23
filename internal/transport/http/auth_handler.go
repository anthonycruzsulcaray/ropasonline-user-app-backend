package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application/auth"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/domain/user"
	"github.com/gin-gonic/gin"
)

const userUUIDContextKey = "authenticatedUserUUID"

type AuthService interface {
	Register(context.Context, string, string, string, string) (user.User, error)
	Login(context.Context, string, string) (auth.LoginResult, error)
	Profile(context.Context, string) (user.User, error)
	UpdateProfile(context.Context, string, string, string, string) (user.User, error)
	DeleteProfile(context.Context, string) error
	Authenticate(context.Context, string) (string, error)
}

type Handler struct {
	service     AuthService
	tokenExpiry time.Duration
}

func RegisterRoutes(router *gin.Engine, service AuthService, tokenExpiry time.Duration) {
	handler := &Handler{service: service, tokenExpiry: tokenExpiry}
	authRoutes := router.Group("/api/v1/auth")
	authRoutes.POST("/register", handler.register)
	authRoutes.POST("/login", handler.login)

	userRoutes := router.Group("/api/v1/users")
	userRoutes.Use(handler.authenticate())
	userRoutes.GET("/me", handler.getProfile)
	userRoutes.PUT("/me", handler.updateProfile)
	userRoutes.DELETE("/me", handler.deleteProfile)
}

type registerRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	LastName string `json:"last_name" binding:"required,min=1,max=100"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=1,max=72"`
}

type updateProfileRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	LastName string `json:"last_name" binding:"required,min=1,max=100"`
	Email    string `json:"email" binding:"required,email,max=254"`
}

type userResponse struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

func (h *Handler) register(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	created, err := h.service.Register(c.Request.Context(), request.Name, request.LastName, request.Email, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, auth.ErrInvalidInput), errors.Is(err, user.ErrInvalidUser):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user data"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not register user"})
		}
		return
	}
	c.JSON(http.StatusCreated, toUserResponse(created))
}

func (h *Handler) login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": result.Token,
		"token_type":   "Bearer",
		"expires_in":   int64(h.tokenExpiry.Seconds()),
		"user":         toUserResponse(result.User),
	})
}

func (h *Handler) getProfile(c *gin.Context) {
	profile, err := h.service.Profile(c.Request.Context(), currentUserUUID(c))
	if err != nil {
		writeUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(profile))
}

func (h *Handler) updateProfile(c *gin.Context) {
	var request updateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	updated, err := h.service.UpdateProfile(c.Request.Context(), currentUserUUID(c), request.Name, request.LastName, request.Email)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case errors.Is(err, user.ErrInvalidUser):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user data"})
		default:
			writeUserError(c, err)
		}
		return
	}
	c.JSON(http.StatusOK, toUserResponse(updated))
}

func (h *Handler) deleteProfile(c *gin.Context) {
	if err := h.service.DeleteProfile(c.Request.Context(), currentUserUUID(c)); err != nil {
		writeUserError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userUUID, err := h.service.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not authenticate request"})
			return
		}
		c.Set(userUUIDContextKey, userUUID)
		c.Next()
	}
}

func currentUserUUID(c *gin.Context) string {
	userUUID, _ := c.Get(userUUIDContextKey)
	return userUUID.(string)
}

func toUserResponse(found user.User) userResponse {
	var updatedAt *string
	if found.UpdatedAt != nil {
		formatted := found.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00")
		updatedAt = &formatted
	}
	return userResponse{
		UUID:      found.UUID,
		Name:      found.Name,
		LastName:  found.LastName,
		Email:     found.Email,
		CreatedAt: found.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt: updatedAt,
	}
}

func writeUserError(c *gin.Context, err error) {
	if errors.Is(err, user.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process user request"})
}
