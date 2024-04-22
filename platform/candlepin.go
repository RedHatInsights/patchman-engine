package platform

import (
	"app/base/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func candlepinHandler(c *gin.Context) {
	envID := c.Param("envid")
	/*
		jsonData, _ := io.ReadAll(c.Request.Body)
		json.Unmarshal(jsonData, &body) // nolint:errcheck
		if body.ReturnStatus > 200 {
			c.AbortWithStatus(body.ReturnStatus)
			return
		}
	*/
	data := fmt.Sprintf(`{
        "environment": "%s"
    }`, envID)
	utils.LogInfo(data)
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func initCandlepin(app *gin.Engine) {
	app.POST("/candlepin/environments/:envid/consumers", candlepinHandler)
}
