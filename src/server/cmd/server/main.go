package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/lazarnagulov/oblak/server/internal/config"
	"github.com/lazarnagulov/oblak/server/internal/platform/logger"
	"go.uber.org/zap"
)

// @title Oblak Serverless API
// @version 1.0
// @description This is the API for the Oblak Serverless platform.

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @host localhost:8080
// @BasePath /api/v1
func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("warning: .env file not found")
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	log, err := logger.New(env)
	if err != nil {
		panic("Failed to create logger: " + err.Error())
	}
	defer log.Sync()

	cfg := config.Load(log)
	cfg.Env = env

	app, err := setupDependencies(cfg, log)
	if err != nil {
		log.Fatal("Failed to setup dependencies", zap.Error(err))
	}
	defer app.DB.Close()

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: app.Router,
	}

	log.Info("starting server",
		zap.String("port", cfg.Port),
		zap.String("environment", cfg.Env),
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
