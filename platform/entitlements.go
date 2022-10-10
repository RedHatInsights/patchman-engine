package platform

import (
	"app/base/types/entitlements"
	"app/base/utils"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

var smartManagement = `{
	"smart_management": {
		"is_entitled": true,
		"is_trial": false
	}
}`

func mockEntitlementsHandler(c *gin.Context) {
	utils.Log().Info("Mocking entitlements api")
	resp := entitlements.Response{}
	if err := json.Unmarshal([]byte(smartManagement), &resp); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, &resp)
}

func initEntitlements(app *gin.Engine) {
	// Mock entitlements api
	app.GET("/api/entitlements/v1/services", mockEntitlementsHandler)
}
