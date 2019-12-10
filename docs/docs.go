package docs

import (
	"github.com/gin-gonic/gin"
)

func HandleOpenapiSpec(c *gin.Context) {
	c.Status(200)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File("./docs/openapi.json")
}

func Init(app *gin.Engine) {
	app.GET("/docs/openapi.json", HandleOpenapiSpec)
}
