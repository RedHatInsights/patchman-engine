package controllers

import (
	"app/base/database"
	"github.com/gin-gonic/gin"
	"net/http"
)

// @Summary	Status endpoint
// @Success 200 {int}		http.StatusOK
// @Failure 503 {object} 	utils.ErrorResponse
func Status(c *gin.Context) {
	sqlDB, _ := database.Db.DB()
	if err := sqlDB.Ping(); err != nil {
		LogAndRespStatusError(c, http.StatusServiceUnavailable, err, "Database not connected")
	} else {
		c.Status(http.StatusOK)
	}
}
