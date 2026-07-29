package platform

import (
	"app/base/content_sources"
	"net/http"

	"github.com/gin-gonic/gin"
)

func templateAdvisoryIDsGetHandler(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "40499999-9999-9999-9999-999999999404" {
		c.Data(http.StatusNotFound, gin.MIMEJSON, []byte{})
		return
	}
	response := content_sources.TemplateAdvisoryIDsResponse{
		AdvisoryIDs: []string{"RH-1", "RH-3"},
	}
	c.JSON(http.StatusOK, response)
}

func initContentSources(app *gin.Engine) {
	// Mock endpoint for content sources
	app.GET("/api/content-sources/v1/templates/:uuid/advisories/ids", templateAdvisoryIDsGetHandler)
}
