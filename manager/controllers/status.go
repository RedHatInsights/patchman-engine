package controllers

import (
	"app/base/database"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Status(c *gin.Context) {
	if err := database.Db.DB().Ping(); err != nil {
		LogAndRespStatusError(c, http.StatusServiceUnavailable, err, "Database not connected")
	} else {
		c.Status(http.StatusOK)
	}
}
