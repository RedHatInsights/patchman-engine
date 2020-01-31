package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func runDebugAPI() {
	app := gin.New()
	app.GET("/sync", func(c *gin.Context) {
		utils.Log().Info("manual syncing called...")
		err := syncAdvisories()
		if err != nil {
			utils.Log("err", err.Error()).Error("manually called syncing failed")
			c.JSON(http.StatusInternalServerError, "error")
			return
		}
		utils.Log().Info("manual syncing finished successfully")
		c.JSON(http.StatusOK, "OK")
	})

	err := app.Run(":9999")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
