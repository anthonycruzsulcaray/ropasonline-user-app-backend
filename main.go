package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/application/auth"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/config"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/mysql"
	"github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/infrastructure/security"
	httpapi "github.com/anthonycruzsulcaray/ropasonline-user-app-backend/internal/transport/http"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	log.Print("archivo .env cargado")

	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	log.Print("configuración validada")

	db, err := sql.Open("mysql", settings.MySQLDSN)
	if err != nil {
		return fmt.Errorf("open MySQL connection: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStartup()
	if err := db.PingContext(startupContext); err != nil {
		return fmt.Errorf("connect to MySQL: %w", err)
	}
	log.Print("conexión a MySQL establecida")

	tokenManager, err := security.NewJWTManager(settings.JWTSecret, settings.JWTDuration)
	if err != nil {
		return fmt.Errorf("configure JWT: %w", err)
	}
	userRepository := mysql.NewUserRepository(db)
	authService := auth.NewService(userRepository, security.BcryptPasswordManager{}, tokenManager)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"mensaje": "pong"})
	})
	httpapi.RegisterRoutes(router, authService, settings.JWTDuration)
	for _, route := range router.Routes() {
		log.Printf("  %s %s", route.Method, route.Path)
	}

	server := &http.Server{
		Addr:              settings.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	log.Printf("API escuchando en http://%s", listener.Addr())

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-shutdownContext.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}
