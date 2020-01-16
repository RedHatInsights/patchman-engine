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

func initRouterWithPath(handler gin.HandlerFunc, path string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator("0"))
	router.GET(path, handler)
	return router
}

func initRouterWithAccount(handler gin.HandlerFunc, path string, account string) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	router.GET(path, handler)
	return router
}

func ParseReponseBody(t *testing.T, bytes []byte, out interface{}) {
	err := json.Unmarshal(bytes, out)
	assert.Nil(t, err)
}
