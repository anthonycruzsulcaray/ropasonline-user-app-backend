package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-gonic/gin"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/cognito"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/config"
	httpapi "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/http"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("get working directory: %v", err)
	}
	if err := config.LoadDotEnvFrom(workingDir); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}

	cognitoClient := cognito.New(awsCfg, cfg.CognitoClientID, cfg.CognitoClientSecret)
	authService := application.NewAuthService(cognitoClient)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	httpapi.RegisterRoutes(router, authService)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shut down HTTP server: %v", err)
		}
	}
}
