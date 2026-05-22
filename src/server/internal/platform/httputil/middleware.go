package httputil

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

type TokenAuthenticator interface {
	AuthenticateToken(ctx context.Context, token string) (int, error)
}

func RequireAPIKey(authService TokenAuthenticator, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			AbortWithError(c, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			AbortWithError(c, http.StatusUnauthorized, "Invalid Authorization format")
			return
		}

		token := parts[1]

		userID, err := authService.AuthenticateToken(c.Request.Context(), token)
		if err != nil {
			log.Warn("Unauthorized access attempt", zap.Error(err), zap.String("ip", c.ClientIP()))
			AbortWithError(c, http.StatusUnauthorized, "Invalid API Key")
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func NewRateLimiterHandler(
	store limiter.RateLimiter,
	limits map[string]limiter.RateLimiterConfig,
	log *zap.Logger,
) func(endpointName string) gin.HandlerFunc {
	return func(endpointName string) gin.HandlerFunc {
		cfg, ok := limits[endpointName]
		if !ok {
			cfg = limits["default"]
			log.Warn("Rate limit config missing, using default", zap.String("endpoint", endpointName))
		}

		return rateLimit(store, endpointName, cfg, log)
	}
}

func rateLimit(limiter limiter.RateLimiter, endpoint string, cfg limiter.RateLimiterConfig, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var identifer string
		if userID, exists := c.Get("userID"); exists {
			identifer = fmt.Sprintf("user:%v", userID)
		} else {
			identifer = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		key := fmt.Sprintf("%s_%s", endpoint, identifer)

		allowed, err := limiter.Allow(c, key, cfg)
		if err != nil {
			log.Warn(
				"Rate limiter temporarily unavailable, request may not be limited",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)
			c.Next()
			return
		}

		if !allowed {
			AbortWithError(c, http.StatusTooManyRequests, "Rate limit exceeded for this action")
			return
		}

		c.Next()
	}
}

func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		requestID := c.GetHeader("X-Request-ID")

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", userAgent),
			zap.String("request_id", requestID),
		}

		switch {
		case status >= 500:
			log.Error("server error", fields...)
		case status >= 400:
			log.Warn("client error", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}
