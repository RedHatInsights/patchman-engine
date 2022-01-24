package middlewares

import (
	"app/docs"
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"
)

func SetSwagger(app *gin.Engine, config docs.EndpointsConfig) {
	// Serving openapi docs
	docs.Init(app, config)

	url := ginSwagger.URL(docs.OpenapiURL)
	app.GET("/openapi/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}
