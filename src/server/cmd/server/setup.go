package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"github.com/lazarnagulov/oblak/server/internal/config"
	"github.com/lazarnagulov/oblak/server/internal/deployment"
	"github.com/lazarnagulov/oblak/server/internal/platform/db"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

type Application struct {
	DB     *sql.DB
	Router http.Handler
}

func setupDependencies(cfg *config.AppConfig, log *zap.Logger) (*Application, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	minioStorage, err := deployment.NewMinioStorage(cfg.Minio, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio storage: %w", err)
	}

	database, err := db.NewPostgresConnection(cfg.DB, log)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	limiter := limiter.NewInMemoryLimiter()
	limitHandler := httputil.NewRateLimiterHandler(limiter, cfg.RateLimits, log)

	r := NewRouter(log)

	authRepo := auth.NewRepository(database)
	authService := auth.NewService(authRepo, log)
	authHandler := auth.NewHandler(authService, limitHandler, log)
	r.RegisterRoutes(authHandler)

	deploymentRepo := deployment.NewRepository(database)
	orchestratorClient := deployment.NewOrchestratorClient(cfg.Orchestrator)
	accessTokenTTL := time.Duration(cfg.AccessURLTTLMinutes) * time.Minute
	deploymentService := deployment.NewService(minioStorage, deploymentRepo, orchestratorClient, accessTokenTTL, log)
	apiURL := cfg.ApiURL
	deploymentHandler := deployment.NewHandler(deploymentService, authService, limitHandler, apiURL, log)
	r.RegisterRoutes(deploymentHandler)

	return &Application{
		DB:     database,
		Router: r.Build(),
	}, nil
}
