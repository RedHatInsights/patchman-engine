package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary	Status endpoint
// @Success 200 {int}		http.StatusOK
// @Failure 503 {object} 	utils.ErrorResponse
func Status(c *gin.Context) {
	sqlDB, err := middlewares.DBWriter(c).DB()
	if err != nil {
		utils.LogAndRespStatusError(c, http.StatusServiceUnavailable, err, "Database not connected")
		return
	}
	if err := sqlDB.Ping(); err != nil {
		utils.LogAndRespStatusError(c, http.StatusServiceUnavailable, err, "Database not connected")
	} else {
		c.Status(http.StatusOK)
	}
}
