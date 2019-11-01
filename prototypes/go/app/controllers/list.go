package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"gin-container/app/database"
	"gin-container/app/structures"
	"gin-container/app/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

func ListHandler(c *gin.Context) {
	var records []structures.HostDAO
	err := database.Db.Model(&structures.HostDAO{}).Order("id DESC").Find(&records).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err":  err.Error()})
		return
	}

	c.JSON(http.StatusOK, &records)
	return
}

func GetHostHandler(c *gin.Context) {
	id, err := utils.LoadParamInt(c, "id", 0, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "wrong param 'id' value"})
		return
	}

	var host structures.HostDAO
	err = database.Db.Where(&structures.HostDAO{ID: id}).First(&host).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"err":  err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"err":  err.Error()})
		return
	}

	// check checksum
	cont := []byte(host.Request)
	bytes := sha256.Sum256(cont)
	computedChecksum := hex.EncodeToString(bytes[:])
	if computedChecksum != host.Checksum {
		c.JSON(http.StatusPartialContent, gin.H{"message": "wrong checksum"})
		return
	}

	c.JSON(http.StatusOK, &host)
	return
}
