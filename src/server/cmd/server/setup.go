package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"github.com/lazarnagulov/oblak/server/internal/config"
	"github.com/lazarnagulov/oblak/server/internal/deployment"
	"github.com/lazarnagulov/oblak/server/internal/platform/db"
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
	_ = minioStorage

	database, err := db.NewPostgresConnection(cfg.DB, log)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	r := NewRouter(log)

	authRepo := auth.NewRepository(database)
	authService := auth.NewService(authRepo, log)
	authHandler := auth.NewHandler(authService, log)
	r.RegisterRoutes(authHandler)

	return &Application{
		DB:     database,
		Router: r.Build(),
	}, nil
}
