package platform

import (
	"app/base/utils"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func candlepinConsumersHandler(c *gin.Context) {
	consumer := c.Param("consumer")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("consumer", consumer, "body", string(jsonData))
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func candlepinConsumersEnvironmentsHandler(c *gin.Context) {
	owner := c.Param("owner")
	jsonData, _ := io.ReadAll(c.Request.Body)
	utils.LogInfo("owner", owner, "body", string(jsonData))
	c.Data(http.StatusOK, gin.MIMEJSON, []byte{})
}

func initCandlepin(app *gin.Engine) {
	app.PUT("/candlepin/consumers/:consumer", candlepinConsumersHandler)
	app.PUT("/candlepin/owner/:owner/consumers/environments", candlepinConsumersEnvironmentsHandler)
}
