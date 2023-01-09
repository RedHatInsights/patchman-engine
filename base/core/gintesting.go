package core

import (
	"app/base/database"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

func InitRouter(handler gin.HandlerFunc) *gin.Engine {
	return InitRouterWithPath(handler, "/")
}

func InitRouterWithParams(handler gin.HandlerFunc, account int, method, path string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	if database.Db != nil {
		router.Use(middlewares.DatabaseWithContext())
	}
	router.Handle(method, path, handler)
	return router
}

func InitRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	return InitRouterWithParams(handler, 1, "GET", path)
}

func InitRouterWithAccount(handler gin.HandlerFunc, path string, account int) *gin.Engine {
	return InitRouterWithParams(handler, account, "GET", path)
}
