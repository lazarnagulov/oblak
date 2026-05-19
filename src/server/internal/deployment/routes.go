package deployment

import (
	"github.com/gin-gonic/gin"
	httputil "github.com/lazarnagulov/oblak/server/internal/platform/httputil"
)

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	funcs := rg.Group("functions")
	funcs.Use(httputil.RequireAPIKey(h.authService, h.log))
	{
		funcs.POST("/", h.Deploy)
		// funcs.GET("/", h.List)
		// funcs.GET("/:name", h.Describe)
		// funcs.DELETE("/:name", h.Delete)
		// funcs.POST("/:name/invoke", h.Invoke)
	}
}
