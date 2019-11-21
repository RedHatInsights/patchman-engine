package manager

import (
	"app/base/utils"
	"app/manager/middlewares"
	"app/manager/routes"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

func RunManager() {
	utils.Log().Info("Manager starting")
	// create web app
	app := gin.New()

	// middlewares
	prometheus := ginprometheus.NewPrometheus("gin")
	prometheus.Use(app)
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	app.HandleMethodNotAllowed = true

	// routes
	routes.Init(app)

	err := app.Run(":8080")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}
