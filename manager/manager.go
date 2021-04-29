package manager

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// nolint: lll
// @title Patchman-engine API
// DO NOT EDIT version MANUALLY - this variable is modified by generate_docs.sh
// @version  v1.10.0
// @description API of the Patch application on [cloud.redhat.com](cloud.redhat.com)
// @description
// @description Syntax of the `filter[name]` query parameters is described in  [Filters documentation](https://github.com/RedHatInsights/patchman-engine/wiki/API-custom-filters)

// @license.name GPLv3
// @license.url https://www.gnu.org/licenses/gpl-3.0.en.html

// @query.collection.format multi
// @securityDefinitions.apikey RhIdentity
// @in header
// @name x-rh-identity
func RunManager() {
	core.ConfigureApp()

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

	err := utils.RunServer(base.Context, app, ":8080")
	if err != nil {
		utils.Log("err", err.Error()).Fatal("server listening failed")
		panic(err)
	}
	utils.Log().Info("manager completed")
}
