package platform

import (
	"app/base/utils"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func candlepinEnvHandler(c *gin.Context) {
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

func candlepinConsumersHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("consumer", consumer, "body", string(jsonData))
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func initCandlepin(app *gin.Engine) {
	app.POST("/candlepin/environments/:envid/consumers", candlepinEnvHandler)
	app.PUT("/candlepin/consumers/:consumer", candlepinConsumersHandler)
}
