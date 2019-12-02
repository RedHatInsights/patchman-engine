package docs

import (
	"github.com/gin-gonic/gin"
)

func Init(app *gin.Engine) {
	app.StaticFile("/docs/openapi.json", "./docs/openapi.json")
}
