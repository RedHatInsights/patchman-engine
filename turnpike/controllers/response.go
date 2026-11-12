package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func respondInternalError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
}
