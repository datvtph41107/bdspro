package web

import "github.com/gin-gonic/gin"

type Router struct {
	engine *gin.Engine
}

func NewRouter() *Router {
	return &Router{
		engine: gin.New(),
	}
}
