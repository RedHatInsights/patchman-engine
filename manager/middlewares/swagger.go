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

func SetAdminSwagger(app *gin.Engine) {
	oaURL := docs.InitAdminAPI((app))

	url := ginSwagger.URL(oaURL)
	api := app.Group("/api/patch/admin")
	api.GET("/openapi/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}
