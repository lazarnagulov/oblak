package deployment

import (
	"github.com/gin-gonic/gin"
	httputil "github.com/lazarnagulov/oblak/server/internal/platform/httputil"
)

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	funcs := rg.Group("/functions")
	funcs.Use(httputil.RequireAPIKey(h.authService, h.log))
	{
		funcs.POST("/", h.rateLimit("deploy"), h.Deploy)
		funcs.GET("/", h.rateLimit("list_functions"), h.List)
		funcs.GET("/:name", h.rateLimit("describe_function"), h.Describe)
		funcs.DELETE("/:name", h.rateLimit("delete_function"), h.Delete)
		// funcs.POST("/:name/invoke", h.Invoke)
	}

	rg.GET("/execute/:token", h.rateLimit("execute_function"), h.ExecuteByToken)
}
