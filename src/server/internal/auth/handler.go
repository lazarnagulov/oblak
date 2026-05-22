package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
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

// Login authenticates a user and returns a token.
// @Summary User login
// @Description Authenticates a user using credentials and returns an API token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User credentials"
// @Success 200 {object} LoginResponse "Successful login"
// @Failure 400 {object} httputil.ErrorResponse  "Invalid request payload"
// @Failure 401 {object} httputil.ErrorResponse  "Invalid credentials"
// @Failure 429 {object} httputil.ErrorResponse  "Rate limit exceeded"
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.service.Login(c.Request.Context(), req.Username, req.Password, "CLIENT")
	if err != nil {
		httputil.WriteError(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		Username: req.Username,
	})

}
