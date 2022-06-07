package routes

import (
	"app/docs"
	"app/manager/controllers"
	"app/manager/middlewares"
	"strings"

	"github.com/gin-gonic/gin"
)

func InitAPI(api *gin.RouterGroup, config docs.EndpointsConfig) { // nolint: funlen
	api.Use(middlewares.RBAC())
	api.Use(middlewares.PublicAuthenticator())
	basePath := api.BasePath()

	advisories := api.Group("/advisories")
	advisories.GET("/", controllers.AdvisoriesListHandler)
	controllers.PreloadAdvisoryCacheItems()
	switch {
	case strings.Contains(basePath, "v1"):
		advisories.GET("/:advisory_id", controllers.AdvisoryDetailHandlerV1)
	case strings.Contains(basePath, "v2"):
		advisories.GET("/:advisory_id", controllers.AdvisoryDetailHandlerV2)
	}
	advisories.GET("/:advisory_id/systems", controllers.AdvisorySystemsListHandler)

	if config.EnableBaselines {
		baselines := api.Group("/baselines")
		baselines.GET("/", controllers.BaselinesListHandler)
		baselines.GET("/:baseline_id", controllers.BaselineDetailHandler)
		baselines.GET("/:baseline_id/systems", controllers.BaselineSystemsListHandler)
		baselines.PUT("/", controllers.CreateBaselineHandler)
		baselines.PUT("/:baseline_id", controllers.BaselineUpdateHandler)
		baselines.DELETE("/:baseline_id", controllers.BaselineDeleteHandler)
		baselines.POST("/systems/remove", controllers.BaselineSystemsRemoveHandler)
	}

	systems := api.Group("/systems")
	systems.GET("/", controllers.SystemsListHandler)
	systems.GET("/:inventory_id", controllers.SystemDetailHandler)
	systems.GET("/:inventory_id/advisories", controllers.SystemAdvisoriesHandler)
	systems.GET("/:inventory_id/packages", controllers.SystemPackagesHandler)
	systems.DELETE("/:inventory_id", controllers.SystemDeleteHandler)

	packages := api.Group("/packages")
	packages.GET("/", controllers.PackagesListHandler)
	packages.GET("/:package_name/systems", controllers.PackageSystemsListHandler)
	packages.GET("/:package_name/versions", controllers.PackageVersionsListHandler)
	packages.GET("/:package_name", controllers.PackageDetailHandler)

	export := api.Group("export")
	export.GET("/advisories", controllers.AdvisoriesExportHandler)
	export.GET("/advisories/:advisory_id/systems", controllers.AdvisorySystemsExportHandler)

	export.GET("/systems", controllers.SystemsExportHandler)
	export.GET("/systems/:inventory_id/advisories", controllers.SystemAdvisoriesExportHandler)
	export.GET("/systems/:inventory_id/packages", controllers.SystemPackagesExportHandler)

	export.GET("/packages", controllers.PackagesExportHandler)
	export.GET("/packages/:package_name/systems", controllers.PackageSystemsExportHandler)

	views := api.Group("/views")
	views.POST("/systems/advisories", controllers.PostSystemsAdvisories)
	views.POST("/advisories/systems", controllers.PostAdvisoriesSystems)

	api.GET("/status", controllers.Status)
	initAdmin(api.Group("/admin"))
}

func initAdmin(group *gin.RouterGroup) {

}
