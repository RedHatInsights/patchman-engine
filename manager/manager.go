package manager

import (
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// @title Patchman-engine API
// @version 1.0
// @description Description here

// @license.name GPLv3
// @license.url https://www.gnu.org/licenses/gpl-3.0.en.html

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
	core.InitProbes(app)
	api := app.Group("/api/patch/v1")
	routes.InitAPI(api)

	err := app.Run(":8080")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
