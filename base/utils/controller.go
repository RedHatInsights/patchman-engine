package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	LogError("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: respMsg})
}

func LogWarnAndResp(c *gin.Context, code int, respMsg string) {
	LogWarn(respMsg)
	c.AbortWithStatusJSON(code, ErrorResponse{Error: respMsg})
}

func LogAndRespStatusError(c *gin.Context, code int, err error, msg string) {
	LogError("err", err.Error(), msg)
	c.AbortWithStatusJSON(code, ErrorResponse{Error: msg})
}

func LogAndRespBadRequest(c *gin.Context, err error, respMsg string) {
	LogWarn("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: respMsg})
}

func LogAndRespNotFound(c *gin.Context, err error, respMsg string) {
	LogWarn("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: respMsg})
}
