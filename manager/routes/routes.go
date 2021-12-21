package routes

import (
	"app/manager/controllers"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

func InitAPI(api *gin.RouterGroup) {
	api.Use(middlewares.RBAC())
	api.Use(middlewares.PublicAuthenticator())

	advisories := api.Group("/advisories")
	advisories.GET("/", controllers.AdvisoriesListHandler)
	controllers.PreloadAdvisoryCacheItems()
	advisories.GET("/:advisory_id", controllers.AdvisoryDetailHandler)
	advisories.GET("/:advisory_id/systems", controllers.AdvisorySystemsListHandler)

	baselines := api.Group("/baselines")
	baselines.GET("/", controllers.BaselinesListHandler)
	baselines.GET("/:baseline_id/systems", controllers.BaselineSystemsListHandler)
	baselines.PUT("/", controllers.CreateBaselineHandler)
	baselines.POST("/:baseline_id", controllers.BaselineUpdateHandler)

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
