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

var basepaths = []string{"/api/patch/v1", "/api/patch/v2", "/api/patch/v3"}

// nolint: lll
// @title Patchman-engine API
// @version  {{.Version}}
// @description API of the Patch application on [console.redhat.com](https://console.redhat.com)
// @description
// @description Syntax of the `filter[name]` query parameters is described in  [Filters documentation](https://github.com/RedHatInsights/patchman-engine/wiki/API-custom-filters)

// @license.name GPLv3
// @license.url https://www.gnu.org/licenses/gpl-3.0.en.html

// @query.collection.format multi
// @securityDefinitions.apikey RhIdentity
// @in header
// @name x-rh-identity

// @BasePath /api/patch/v3
func RunManager() {
	core.ConfigureApp()

	port := utils.Cfg.PublicPort
	utils.LogInfo("port", port, "Manager starting at port")
	// create web app
	app := gin.New()

	// middlewares
	app.Use(gin.Recovery())
	middlewares.Prometheus().Use(app)
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	app.Use(middlewares.WithTimeout(utils.Cfg.ResponseTimeout))
	endpointsConfig := getEndpointsConfig()
	middlewares.SetSwagger(app, endpointsConfig)
	app.HandleMethodNotAllowed = true

	// routes
	core.InitProbes(app)
	for _, path := range basepaths {
		api := app.Group(path)
		routes.InitAPI(api, endpointsConfig)
	}

	go base.TryExposeOnMetricsPort(app)

	kafka.TryStartEvalQueue(mqueue.NewKafkaWriterFromEnv)

	err := utils.RunServer(base.Context, app, port)
	if err != nil {
		utils.LogFatal("err", err.Error(), "server listening failed")
		panic(err)
	}
	utils.LogInfo("manager completed")
}

func getEndpointsConfig() docs.EndpointsConfig {
	config := docs.EndpointsConfig{
		EnableBaselines: utils.GetBoolEnvOrDefault("ENABLE_BASELINES_API", true),
	}
	return config
}
