package manager

import (
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	_ "app/docs"
)


func RunManager() {
	utils.Log().Info("Manager starting")
	// create web app
	app := gin.New()

	// middlewares
	middlewares.Prometheus().Use(app)
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	app.HandleMethodNotAllowed = true

	url := ginSwagger.URL("http://localhost:8080/swagger/doc.json") // The url pointing to API definition
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	// routes
	routes.Init(app)

	api := app.Group("/api/patch/v1")
	routes.InitAPI(api)

	err := app.Run(":8080")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
