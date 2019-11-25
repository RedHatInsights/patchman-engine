package controllers

import (
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

func initRouter(handler gin.HandlerFunc) *gin.Engine {
	return initRouterWithPath(handler, "/")
}

func initRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.GET(path, handler)
	return router
}
