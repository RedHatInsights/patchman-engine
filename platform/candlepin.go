package platform

import (
	"app/base/candlepin"
	"app/base/utils"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	if envID == "return_404" {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func candlepinConsumersPutHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("PUT consumer", consumer, "body", string(jsonData))
	if consumer == "return_404" {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func candlepinConsumersGetHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	utils.LogInfo("GET consumer", consumer, "body")
	env := strings.ReplaceAll(consumer, "-", "")
	env = strings.Replace(env, "000", "999", 1)
	response := candlepin.ConsumersDetailResponse{
		Environments: []candlepin.ConsumersEnvironment{
			{ID: env},
		},
	}
	c.JSON(http.StatusOK, response)
}

func initCandlepin(app *gin.Engine) {
	app.POST("/candlepin/environments/:envid/consumers", candlepinEnvHandler)
	app.PUT("/candlepin/consumers/:consumer", candlepinConsumersPutHandler)
	app.GET("/candlepin/consumers/:consumer", candlepinConsumersGetHandler)
}
