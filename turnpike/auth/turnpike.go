package auth

import (
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TurnpikeAuthenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		identStr := c.GetHeader("x-rh-identity")
		if identStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing x-rh-identity header"})
			return
		}
		utils.LogTrace("ident", identStr, "Identity retrieved")

		xrhid, err := utils.ParseXRHID(identStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}

		if xrhid.Identity.Type != "associate" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}
	}
}
