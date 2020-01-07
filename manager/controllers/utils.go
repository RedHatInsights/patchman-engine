package controllers

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Error(respMsg)
	c.JSON(http.StatusInternalServerError, utils.ErrorResponse{respMsg})
}

func LogAndRespBadRequest(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.JSON(http.StatusBadRequest, utils.ErrorResponse{respMsg})
}

func LogAndRespNotFound(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.JSON(http.StatusNotFound, utils.ErrorResponse{respMsg})
}
