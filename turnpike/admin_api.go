package turnpike

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"

	"github.com/gin-gonic/gin"
)

// nolint: lll
// @title Patch Admin API
// DO NOT EDIT version MANUALLY - this variable is modified by generate_docs.sh
// @version  v2.3.9
// @description Admin API of the Patch application on [internal.console.redhat.com](https://internal.console.redhat.com)

// @license.name GPLv3
// @license.url https://www.gnu.org/licenses/gpl-3.0.en.html

// @query.collection.format multi
// @securityDefinitions.apikey RhIdentity
// @in header
// @name x-rh-identity

// @BasePath /api/patch/admin
func RunAdminAPI() {
	core.ConfigureApp()

	utils.Log("port", utils.Cfg.PublicPort).Info("Manager-admin starting")
	app := gin.New()
	app.Use(middlewares.RequestResponseLogger())
	middlewares.SetAdminSwagger(app)

	core.InitProbes(app)
	routes.InitAdmin(app)

	err := utils.RunServer(base.Context, app, utils.Cfg.PublicPort)
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
