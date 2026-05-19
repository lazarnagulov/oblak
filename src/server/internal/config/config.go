package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/lazarnagulov/oblak/server/internal/deployment"
	"github.com/lazarnagulov/oblak/server/internal/platform/db"
	"go.uber.org/zap"
)

type AppConfig struct {
	Env   string
	Port  string
	DB    db.Config
	Minio deployment.MinioConfig
}

func Load(log *zap.Logger) *AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, falling back to system environment variables")
	} else {
		log.Info(".env file successfully loaded")
	}

	env := getEnv("APP_ENV", "development")

	return &AppConfig{
		Env:  env,
		Port: getEnv("PORT", "8080"),
		DB: db.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "admin"),
			Password: getEnv("DB_PASSWORD", "securepass2026!"),
			DBName:   getEnv("DB_NAME", "db"),
		},
		Minio: deployment.MinioConfig{
			Endpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:  getEnv("MINIO_ACCESS_KEY", "oblak_admin"),
			SecretKey:  getEnv("MINIO_SECRET_KEY", "oblak_super_secret_password"),
			BucketName: "oblak-artifacts",
			UseSSL:     false,
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
