package middlewares

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"net/http"
	"strings"
	"sync"
)

const KeyAccount = "account"

var AccountIDCache = struct {
	Values map[string]int
	Lock   sync.Mutex
}{Values: map[string]int{}, Lock: sync.Mutex{}}

func findAccount(c *gin.Context, identity *identity.Identity) bool {
	AccountIDCache.Lock.Lock()
	defer AccountIDCache.Lock.Unlock()

	if id, has := AccountIDCache.Values[identity.AccountNumber]; has {
		c.Set(KeyAccount, id)
	} else {
		var acc models.RhAccount
		if err := database.Db.Where("name = ?", identity.AccountNumber).Find(&acc).Error; err != nil {
			c.AbortWithStatus(http.StatusNoContent)
			return false
		}
		AccountIDCache.Values[acc.Name] = acc.ID
		c.Set(KeyAccount, acc.ID)
	}
	return true
}

func PublicAuthenticator() gin.HandlerFunc {
	devModeEnabled := utils.GetBoolEnvOrDefault("ENABLE_DEV_MODE", false)
	if devModeEnabled {
		accountID := utils.GetIntEnvOrDefault("DEV_ACCOUNT_ID", 1)
		return MockAuthenticator(accountID)
	}
	return headerAuthenticator()
}

func headerAuthenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		identStr := c.GetHeader("x-rh-identity")
		if identStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing x-rh-identity header"})
			return
		}
		utils.Log("ident", identStr).Trace("Identity retrieved")

		ident, err := utils.ParseIdentity(identStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}
		if findAccount(c, ident) {
			c.Next()
		}
	}
}

func TurnpikeAuthenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		identStr := c.GetHeader("x-rh-identity")
		if identStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing x-rh-identity header"})
			return
		}
		utils.Log("ident", identStr).Trace("Identity retrieved")
		ident, err := utils.ParseIdentity(identStr)
		// Turnpike endpoints only support associate
		if err != nil || strings.ToLower(ident.Type) != "associate" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
			return
		}
	}
}

func MockAuthenticator(account int) gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Log("account_id", account).Warn("using mocking account id")
		c.Set(KeyAccount, account)
		c.Next()
	}
}
