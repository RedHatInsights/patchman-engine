package core

import (
	"app/base/database"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func Readiness(c *gin.Context) {
	err := database.Db.Exec(`\d`).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "ok")
}

func InitProbes(app *gin.Engine) {
	// public routes
	app.GET("/liveness", Liveness)
	app.GET("/readiness", Readiness)
}
