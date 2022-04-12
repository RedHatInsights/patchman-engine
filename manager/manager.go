package manager

import (
	"app/base"
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"app/docs"
	"app/manager/kafka"
	"app/manager/middlewares"
	"app/manager/routes"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// nolint: lll
// @title Patchman-engine API
// DO NOT EDIT version MANUALLY - this variable is modified by generate_docs.sh
// @version  v1.18.67
// @description API of the Patch application on [console.redhat.com](https://console.redhat.com)
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
	endpointsConfig := getEndpointsConfig()
	middlewares.SetSwagger(app, endpointsConfig)
	app.HandleMethodNotAllowed = true

	// routes
	core.InitProbes(app)
	api := app.Group("/api/patch/v1")
	routes.InitAPI(api, endpointsConfig)

	go base.TryExposeOnMetricsPort(app)

	kafka.TryStartEvalQueue(mqueue.NewKafkaWriterFromEnv)

	port := utils.GetIntEnvOrDefault("PUBLIC_PORT", 8080)
	err := utils.RunServer(base.Context, app, port)
	if err != nil {
		utils.Log("err", err.Error()).Fatal("server listening failed")
		panic(err)
	}
	utils.Log().Info("manager completed")
}

func getEndpointsConfig() docs.EndpointsConfig {
	config := docs.EndpointsConfig{
		EnableBaselines: utils.GetBoolEnvOrDefault("ENABLE_BASELINES_API", true),
	}
	return config
}
