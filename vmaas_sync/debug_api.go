package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/database"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func runDebugAPI() {
	app := gin.New()
	app.GET("/sync", sync)
	app.GET("/re-calc", recalc)
	app.GET("/check-caches", checkCaches)

	err := app.Run(":9999")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func sync(c *gin.Context) {
	utils.Log().Info("manual syncing called...")
	err := syncAdvisories()
	if err != nil {
		utils.Log("err", err.Error()).Error("manual called syncing failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.Log().Info("manual syncing finished successfully")
	c.JSON(http.StatusOK, "OK")
}

func recalc(c *gin.Context) {
	utils.Log().Info("manual re-calc messages sending called...")
	err := sendReevaluationMessages()
	if err != nil {
		utils.Log("err", err.Error()).Error("manual re-calc msgs sending failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.Log().Info("manual re-calc messages sent successfully")
	c.JSON(http.StatusOK, "OK")
}

func checkCaches(c *gin.Context) {
	valid, err := database.CheckCachesValid()
	if err != nil {
		utils.Log("error", err).Error("Could not check validity of caches")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}

	if !valid {
		utils.Log().Error("Cache mismatch found")
		c.JSON(http.StatusConflict, "conflict")
		return
	}

	c.JSON(http.StatusOK, "caches counts OK")
}
