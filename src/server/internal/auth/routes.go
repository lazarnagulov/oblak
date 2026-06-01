package auth

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.rateLimit("login"), h.Login)
	}
}
