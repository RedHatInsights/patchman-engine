package platform

import (
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
)

func runMockInventory() {
	utils.Log().Info("Platform mock starting")
	app := gin.New()
	app.Use(middlewares.RequestResponseLogger())
	Init(app)
	app.Run(":9001")
}
// Function, which mocks platform, returning a valid system_profile
func RunPlatform() {
	runMockInventory()
}
