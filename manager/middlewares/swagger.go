package middlewares

import (
	"app/docs"

	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"
)

func SetSwagger(app *gin.Engine, config docs.EndpointsConfig) {
	// Serving openapi docs
	openapiURL := docs.Init(app, config)

	url := ginSwagger.URL(openapiURL)
	app.GET("/openapi/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}
