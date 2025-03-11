package controllers

import (
	"app/base/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Recreate pg_repack database extension
// @Description Recreate pg_repack database extension
// @ID repackRecreate
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} string
// @Failure 500 {object} map[string]interface{}
// @Router /database/pg_repack/recreate [put]
func RepackRecreateHandler(c *gin.Context) {
	if err := database.DB.Exec("DROP EXTENSION IF EXISTS pg_repack CASCADE").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	if err := database.DB.Exec("CREATE EXTENSION pg_repack").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
	}
	c.JSON(http.StatusOK, "pg_repack extension re-created")
}
