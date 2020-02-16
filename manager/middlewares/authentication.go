package middlewares

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

const KeyAccount = "account"

func Authenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		identStr := c.GetHeader("x-rh-identity")
		if identStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing x-rh-identity header"})
			return
		}
		utils.Log("ident", identStr).Trace("Identity retrieved")

		identity, err := utils.ParseIdentity(identStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}
		c.Set(KeyAccount, identity.Identity.AccountNumber)
		c.Next()
	}
}

func MockAuthenticator(account string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(KeyAccount, account)
		c.Next()
	}
}
