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

// Login authenticates a user and returns a JWT token.
// @Summary User login
// @Description Authenticates a user using credentials and returns an API token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "User credentials"
// @Success 200 {object} auth.LoginResponse "Successful login"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Router /auth/login [post]
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

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		Username: req.Username,
	})

}
