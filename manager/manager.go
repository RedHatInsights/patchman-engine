package manager

import (
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// @title Patchman-engine API
// @version 1.0
// @description Description here

// @securityDefinitions.apikey RhIdentity
// @in header
// @name x-rh-identity
func RunManager() {
	utils.Log().Info("Manager starting")
	// create web app
	app := gin.New()

	// middlewares
	middlewares.Prometheus().Use(app)
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	middlewares.SetSwagger(app)
	app.HandleMethodNotAllowed = true

	// routes
	routes.Init(app)

	api := app.Group("/api/patch/v1")
	routes.InitAPI(api)

	err := app.Run(":8080")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
