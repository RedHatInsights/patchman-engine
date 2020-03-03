package controllers

import (
	"app/manager/middlewares"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"testing"
)

func initRouter(handler gin.HandlerFunc) *gin.Engine {
	return initRouterWithPath(handler, "/")
}

func initRouterWithParams(handler gin.HandlerFunc, account, method, path string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	router.Handle(method, path, handler)
	return router
}

func initRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	return initRouterWithParams(handler, "1", "GET", path)
}

func initRouterWithAccount(handler gin.HandlerFunc, path string, account string) *gin.Engine {
	return initRouterWithParams(handler, account, "GET", path)
}

func ParseReponseBody(t *testing.T, bytes []byte, out interface{}) {
	err := json.Unmarshal(bytes, out)
	assert.Nil(t, err, string(bytes))
}
