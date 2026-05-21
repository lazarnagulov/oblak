package httputil

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

func RequireAPIKey(authService auth.Service, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization format"})
			return
		}

		token := parts[1]

		userID, err := authService.AuthenticateToken(c.Request.Context(), token)
		if err != nil {
			log.Warn("Unauthorized access attempt", zap.Error(err), zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func RateLimit(limiter limiter.RateLimiter, endpoint string, cfg limiter.RateLimiterConfig, log *zap.Logger) gin.HandlerFunc {
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
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded for this action",
			})
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
