package routes

import (
	"app/manager/controllers"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

func InitAPI(group *gin.RouterGroup) {
	group.Use(middlewares.Authenticator())
	group.Use(middlewares.RBAC())

	group.GET("/advisories", controllers.AdvisoriesListHandler)
	group.GET("/advisories/:advisory_id", controllers.AdvisoryDetailHandler)
	group.GET("/advisories/:advisory_id/systems", controllers.AdvisorySystemsListHandler)

	group.GET("/systems", controllers.SystemsListHandler)
	group.GET("/systems/:inventory_id", controllers.SystemDetailHandler)
	group.GET("/systems/:inventory_id/advisories", controllers.SystemAdvisoriesHandler)
	group.GET("/systems/:inventory_id/packages", controllers.SystemPackagesHandler)
	group.DELETE("/systems/:inventory_id", controllers.SystemDeleteHandler)

	group.GET("/packages/:package_name/systems", controllers.PackageSystemsListHandler)
	group.GET("/packages", controllers.PackagesListHandler)
	group.GET("/packages/:package_name", controllers.PackageDetailHandler)

	group.GET("/export/advisories", controllers.AdvisoriesExportHandler)
	group.GET("/export/systems", controllers.SystemsExportHandler)

	group.GET("/status", controllers.Status)
}
