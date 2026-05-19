package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
	"go.uber.org/zap"
)

type Routable interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

type Router struct {
	api []Routable
	log *zap.Logger
}

func NewRouter(log *zap.Logger) *Router {
	return &Router{log: log}
}

func (r *Router) RegisterRoutes(routers ...Routable) {
	r.api = append(r.api, routers...)
}

func (r *Router) Build() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httputil.RequestLogger(r.log))

	api := engine.Group("/api/v1")
	for _, route := range r.api {
		route.RegisterRoutes(api)
	}

	return engine
}
