package controllers

import (
	"gin-container/app/database"
	"gin-container/app/structures"
	"gin-container/app/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func CreateHandler(c *gin.Context) {
	id, err := utils.LoadParamInt(c, "id", 0, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "wrong param 'id' value"})
		return
	}

	record := structures.HostDAO{ID: id, Request: "req", Checksum: "chs"}

	result := database.Db.Save(&record)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err":  result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "result already exits"})
		return
	}

	c.JSON(http.StatusOK, record)
	return
}
