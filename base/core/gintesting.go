package core

import (
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

func InitRouter(handler gin.HandlerFunc) *gin.Engine {
	return InitRouterWithPath(handler, "/")
}

func InitRouterWithParams(handler gin.HandlerFunc, account, method, path string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	router.Handle(method, path, handler)
	return router
}

func InitRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	return InitRouterWithParams(handler, "1", "GET", path)
}

func InitRouterWithAccount(handler gin.HandlerFunc, path string, account string) *gin.Engine {
	return InitRouterWithParams(handler, account, "GET", path)
}
