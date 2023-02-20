package middlewares

import (
	"app/docs"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"
)

const KeyApiver = "apiver"

var apiRegexp = regexp.MustCompile(`/v(\d)/`)

func apiver(path string) int {
	match := apiRegexp.FindStringSubmatch(path)
	if len(match) > 1 {
		i, err := strconv.Atoi(match[1])
		if err == nil {
			return i
		}
	}
	return 1
}

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

func SetAPIVersion(basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(KeyApiver, apiver(basePath))
	}
}
