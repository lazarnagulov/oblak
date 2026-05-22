package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

type Handler struct {
	service   Service
	rateLimit limiter.LimitHandler
	log       *zap.Logger
}

func NewHandler(service Service, rateLimit limiter.LimitHandler, log *zap.Logger) *Handler {
	return &Handler{
		service:   service,
		rateLimit: rateLimit,
		log:       log,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, err := h.service.Login(c.Request.Context(), req.Username, req.Password, "CLIENT")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": req.Username,
	})

}
