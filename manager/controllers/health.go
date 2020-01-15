package controllers

import (
	"app/base/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

func HealthDBHandler(c *gin.Context) {
	err := database.Db.DB().Ping()
	if err != nil {
		c.String(http.StatusInternalServerError, "unable to ping database")
	}

	c.String(http.StatusOK, "OK")
}
