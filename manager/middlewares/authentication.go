package middlewares

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

const KeyAccount = "account"

var AccountIDCache = struct {
	Values map[string]int
	Lock   sync.Mutex
}{Values: map[string]int{}, Lock: sync.Mutex{}}

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
		AccountIDCache.Lock.Lock()
		defer AccountIDCache.Lock.Unlock()

		if id, has := AccountIDCache.Values[identity.Identity.AccountNumber]; has {
			c.Set(KeyAccount, id)
		} else {
			var acc models.RhAccount
			if err := database.Db.Where("name = ?", identity.Identity.AccountNumber).Find(&acc).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Could not find rh_account"})
			}
			AccountIDCache.Values[acc.Name] = acc.ID
			c.Set(KeyAccount, acc.ID)
		}
		c.Next()
	}
}

func MockAuthenticator(account int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(KeyAccount, account)
		c.Next()
	}
}
