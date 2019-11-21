package controllers

import (
	"app/base/database"
	"app/base/structures"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func DeleteHandler(c *gin.Context) {
	id, err := utils.LoadParamInt(c, "id", 0, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "wrong param 'id' value"})
		return
	}

	record := structures.HostDAO{}

	nRowsAffected := database.Db.Where(
		"id = ?", id).Delete(&record).RowsAffected

	if nRowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "no such version found"})
		return
	}

	c.JSON(http.StatusOK, &record)
	return
}
