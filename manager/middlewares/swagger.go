package middlewares

import (
	"app/docs"
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"
)

func SetSwagger(app *gin.Engine) {
	// Serving openapi docs
	docs.Init(app)

	url := ginSwagger.URL("/docs/openapi.json")
	app.GET("/openapi/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}
