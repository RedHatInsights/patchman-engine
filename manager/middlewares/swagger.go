package middlewares

import (
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"

	_ "app/docs"
)

func SetSwagger(app *gin.Engine) {
	url := ginSwagger.URL("http://localhost:8080/swagger/doc.json") // The url pointing to API definition
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}
