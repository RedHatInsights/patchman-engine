package turnpike

import (
	"app/base"
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"

	"github.com/gin-gonic/gin"
)

// @title Patch Admin API
// @version  {{.Version}}
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

	// Toggle Turnpike authentication for internal API (manual sync, re-calc)
	enableTurnpikeAuth := utils.PodConfig.GetBool("turnpike_auth", true)

	utils.LogInfo("port", utils.CoreCfg.PublicPort, "Manager-admin starting")
	app := gin.New()
	app.Use(middlewares.RequestResponseLogger())
	middlewares.SetAdminSwagger(app)

	core.InitProbes(app)
	routes.InitAdmin(app, enableTurnpikeAuth)

	err := utils.RunServer(base.Context, app, utils.CoreCfg.PublicPort)
	if err != nil {
		utils.LogError("err", err.Error())
		panic(err)
	}
}
