package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"github.com/lazarnagulov/oblak/server/internal/platform/db"
	"github.com/lazarnagulov/oblak/server/internal/platform/logger"
	"go.uber.org/zap"
)

func main() {
	log, err := logger.New("production")
	if err != nil {
		log.Panic("Failed to create logger")
	}
	defer log.Sync()

	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, falling back to system environment variables")
	} else {
		log.Info(".env file successfully loaded")
	}

	dbCfg := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "admin"),
		Password: getEnv("DB_PASSWORD", "securepass2026!"),
		DBName:   getEnv("DB_NAME", "db"),
	}

	serverPort := getEnv("PORT", "8080")

	db, err := db.NewPostgresConnection(dbCfg, log)
	if err != nil {
		log.Fatal("Database connection failed", zap.Error(err))
	}
	defer db.Close()

	r := NewRouter(log)

	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, log)
	authHandler := auth.NewHandler(authService, log)
	r.RegisterRoutes(authHandler)
	router := r.Build()

	srv := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	log.Info("starting server",
		zap.String("port", serverPort),
		zap.String("environment", "production"),
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
