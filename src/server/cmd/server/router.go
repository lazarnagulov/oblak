package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lazarnagulov/oblak/server/docs"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Routable interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

type Router struct {
	api []Routable
	log *zap.Logger
	env string
}

func NewRouter(env string, log *zap.Logger) *Router {
	return &Router{env: env, log: log}
}

func (r *Router) RegisterRoutes(routers ...Routable) {
	r.api = append(r.api, routers...)
}

func (r *Router) Build() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httputil.RequestLogger(r.log))

	if r.env != "production" {
		r.log.Info("Swagger documentation enabled")
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := engine.Group("/api/v1")
	for _, route := range r.api {
		route.RegisterRoutes(api)
	}

	return engine
}
