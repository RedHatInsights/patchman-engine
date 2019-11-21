package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"

	"app/base/structures"
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

func createTestingSample(id int) {
	record := &structures.HostDAO{ID: id, Request: "r",
		Checksum: "454349e422f05297191ead13e21d3db520e5abef52055e4964b82fb213f593a1"}
	err := database.Db.Create(record).Error
	if err != nil {
		panic(err)
	}
}
