package platform

import (
	"app/base/candlepin"
	"app/base/utils"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

func candlepinConsumersPutHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("PUT consumer", consumer, "body", string(jsonData))
	if consumer == "return_404" || consumer == "99999999-9999-9999-9999-999999999404" {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func candlepinConsumersGetHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	utils.LogInfo("GET consumer", consumer, "body")
	if consumer == "return_404" || consumer == "99999999-9999-9999-9999-999999999404" {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	env := strings.ReplaceAll(consumer, "-", "")
	env = strings.Replace(env, "000", "999", 1)
	response := candlepin.ConsumersDetailResponse{
		Environments: []candlepin.ConsumersEnvironment{
			{ID: env},
		},
	}
	c.JSON(http.StatusOK, response)
}

func candlepinConsumersEnvironmentsHandler(c *gin.Context) {
	owner := c.Param("owner")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("owner", owner, "body", string(jsonData))
	var req candlepin.ConsumersEnvironmentsRequest
	err := json.Unmarshal(jsonData, &req)
	if err != nil {
		c.Data(http.StatusInternalServerError, gin.MIMEJSON, []byte{})
		return
	}
	utils.LogInfo("ConsumerUuids", req.ConsumerUuids)
	if slices.Contains(req.ConsumerUuids, "return_404") ||
		slices.Contains(req.ConsumerUuids, "99999999-9999-9999-9999-999999999404") {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func initCandlepin(app *gin.Engine) {
	app.PUT("/candlepin/consumers/:consumer", candlepinConsumersPutHandler)
	app.GET("/candlepin/consumers/:consumer", candlepinConsumersGetHandler)
	app.PUT("/candlepin/owners/:owner/consumers/environments", candlepinConsumersEnvironmentsHandler)
}
