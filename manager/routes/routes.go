package routes

import (
	"app/manager/controllers"
	"github.com/gin-gonic/gin"
)

// Init routes.
func Init(app *gin.Engine) {
	// public routes
	app.GET("/health", controllers.HealthHandler)
	app.GET("/db_health", controllers.HealthDBHandler)
	app.GET("/samples", controllers.ListHandler)
	app.GET("/hosts/:id", controllers.GetHostHandler)
	app.GET("/create", controllers.CreateHandler)
	app.GET("/delete", controllers.DeleteHandler)
}
