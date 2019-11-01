package routes

import (
	"github.com/gin-gonic/gin"
	"gin-container/app/controllers"
	"gin-container/app/utils"
)

// Init routes.
func Init(app *gin.Engine) {
	// public routes
	app.GET("/health", controllers.HealthHandler)
	app.GET("/db_health", controllers.HealthDBHandler)
	app.GET("/samples", controllers.ListHandler)
	app.GET("/hosts/:id", controllers.GetHostHandler)

	// private auth required routes
	private := app.Group("/private", gin.BasicAuth(gin.Accounts{
		utils.GetenvOrFail("PRIVATE_API_USER"): utils.GetenvOrFail("PRIVATE_API_PASSWD"),
	}))

	private.GET("/create", controllers.CreateHandler)
	private.GET("/delete", controllers.DeleteHandler)
}
