package turnpike

import (
	"app/base"
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	sync "app/tasks/vmaas_sync"
	"net/http"

	"github.com/gin-gonic/gin"
)

var enableTurnpikeAuth bool

func init() {
	enableTurnpikeAuth = utils.GetBoolEnvOrDefault("ENABLE_TURNPIKE_AUTH", false)
}

func RunAdminAPI() {
	app := gin.New()

	if enableTurnpikeAuth {
		app.Use(middlewares.TurnpikeAuthenticator())
	}

	app.GET("/sync", syncapi)
	app.GET("/re-calc", recalc)
	app.GET("/check-caches", checkCaches)

	err := utils.RunServer(base.Context, app, utils.Cfg.PrivatePort)

	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func syncapi(c *gin.Context) {
	utils.Log().Info("manual syncing called...")
	err := sync.SyncData(nil, nil)
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
	err := sync.SendReevaluationMessages()
	if err != nil {
		utils.Log("err", err.Error()).Error("manual re-calc msgs sending failed")
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	utils.Log().Info("manual re-calc messages sent successfully")
	c.JSON(http.StatusOK, "OK")
}

func checkCaches(c *gin.Context) {
	valid, err := database.CheckCachesValidRet()
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
