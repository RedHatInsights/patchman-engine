package webserver

import (
	"gin-container/app/middlewares"
	"gin-container/app/routes"
	"gin-container/app/utils"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

func RunWebserver() {
	utils.Log().Info("webserver starting")
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
