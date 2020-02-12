package routes

import (
	"app/manager/controllers"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

// Init routes.
func Init(app *gin.Engine) {
	// public routes
	app.GET("/health", controllers.HealthHandler)
	app.GET("/db_health", controllers.HealthDBHandler)
}

func InitAPI(group *gin.RouterGroup) {
	group.Use(middlewares.Authenticator())
	group.Use(middlewares.RBAC())
	group.GET("/advisories", controllers.AdvisoriesListHandler)
	group.GET("/advisories/:advisory_id", controllers.AdvisoryDetailHandler)
	group.GET("/advisories/:advisory_id/systems", controllers.AdvisorySystemsListHandler)
	group.GET("/systems", controllers.SystemsListHandler)
	group.GET("/systems/:inventory_id", controllers.SystemDetailHandler)
	group.GET("/systems/:inventory_id/advisories", controllers.SystemAdvisoriesHandler)
}
