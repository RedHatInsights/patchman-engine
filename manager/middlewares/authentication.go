package middlewares

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

const KEY_ACCOUNT = "account"

func Authenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		identStr := c.GetHeader("x-rh-identity")
		if identStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing x-rh-identity header"})
			return
		}

		identity, err := utils.ParseIdentity(identStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}
		c.Set(KEY_ACCOUNT, identity.Identity.AccountNumber)
		c.Next()
	}
}

func MockAuthenticator(account string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(KEY_ACCOUNT, account)
		c.Next()
	}
}
