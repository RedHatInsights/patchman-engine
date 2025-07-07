package platform

import (
	"app/base/candlepin"
	"app/base/utils"
	"encoding/json"
	"io"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

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
	if slices.Contains(req.ConsumerUuids, "return_404") {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func initCandlepin(app *gin.Engine) {
	app.PUT("/candlepin/consumers/:consumer", candlepinConsumersPutHandler)
	app.PUT("/candlepin/owners/:owner/consumers/environments", candlepinConsumersEnvironmentsHandler)
}
