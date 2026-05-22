package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/lazarnagulov/oblak/server/internal/deployment"
	"github.com/lazarnagulov/oblak/server/internal/platform/db"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

type AppConfig struct {
	Env        string
	Port       string
	DB         db.Config
	Minio      deployment.MinioConfig
	RateLimits map[string]limiter.RateLimiterConfig
}

func Load(log *zap.Logger) *AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, falling back to system environment variables")
	} else {
		log.Info(".env file successfully loaded")
	}

	env := getEnv("APP_ENV", "development")
	rateLimitsPath := getEnv("RATE_LIMITS_FILE", "rate_limits.yaml")

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
			UseSSL:     getEnv("MINIO_USE_SSL", "") == "true",
		},
		RateLimits: loadRateLimits(rateLimitsPath, log),
	}
}

func loadRateLimits(path string, log *zap.Logger) map[string]limiter.RateLimiterConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Warn("Could not read rate limits config file, using safe defaults", zap.String("path", path), zap.Error(err))
		return getDefaultRateLimits()
	}
	var parsed struct {
		RateLimits map[string]limiter.RateLimiterConfig `yaml:"rate_limits"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		log.Error("Failed to parse rate limits YAML, using safe defaults", zap.Error(err))
		return getDefaultRateLimits()
	}

	log.Info("Rate limits successfully loaded from YAML")
	return parsed.RateLimits
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getDefaultRateLimits() map[string]limiter.RateLimiterConfig {
	return map[string]limiter.RateLimiterConfig{
		"default": {Capacity: 10, Refill: 1.0},
		"login":   {Capacity: 5, Refill: 0.05},
		"deploy":  {Capacity: 5, Refill: 0.1},
	}
}
