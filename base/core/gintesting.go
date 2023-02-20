package core

import (
	"app/base/database"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

type ContextKV struct {
	Key   string
	Value any
}

func InitRouter(handler gin.HandlerFunc, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithPath(handler, "/", contextKVs...)
}

func InitRouterWithParams(handler gin.HandlerFunc, account int, method, path string,
	contextKVs ...ContextKV) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	if database.Db != nil {
		router.Use(middlewares.DatabaseWithContext())
	}
	router.Use(func(c *gin.Context) {
		for _, kv := range contextKVs {
			c.Set(kv.Key, kv.Value)
		}
	})
	router.Handle(method, path, handler)
	return router
}

func InitRouterWithPath(handler gin.HandlerFunc, path string, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithParams(handler, 1, "GET", path, contextKVs...)
}

func InitRouterWithAccount(handler gin.HandlerFunc, path string, account int, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithParams(handler, account, "GET", path, contextKVs...)
}
