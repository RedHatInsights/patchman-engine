package core

import (
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

func InitRouter(handler gin.HandlerFunc) *gin.Engine {
	return InitRouterWithPath(handler, "/")
}

func InitRouterWithExtraParams(handler gin.HandlerFunc, account int, method, path string, timeoutMs int) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	if timeoutMs > 0 {
		router.Use(middlewares.Timeout(timeoutMs))
	}
	router.Handle(method, path, handler)
	return router
}

func InitRouterWithParams(handler gin.HandlerFunc, account int, method, path string) *gin.Engine {
	router := InitRouterWithExtraParams(handler, account, method, path, 0)
	return router
}

func InitRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	return InitRouterWithParams(handler, 1, "GET", path)
}

func InitRouterWithAccount(handler gin.HandlerFunc, path string, account int) *gin.Engine {
	return InitRouterWithParams(handler, account, "GET", path)
}

func InitRouterWithTimeout(handler gin.HandlerFunc, timeoutMs int) *gin.Engine {
	return InitRouterWithExtraParams(handler, 1, "GET", "/", timeoutMs)
}
