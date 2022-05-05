package middlewares

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const KeyAccount = "account"

var AccountIDCache = struct {
	Values map[string]int
	Lock   sync.Mutex
}{Values: map[string]int{}, Lock: sync.Mutex{}}

// Stores or updates the account data, returning the account id
func GetOrCreateAccount(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}
	// Select, and only if not found attempt an insertion
	err := database.Db.Where("name = ?", account).Find(&rhAccount).Error
	if err != nil {
		utils.Log("err", err, "name", account).Warn("Error in finding account")
	}
	if rhAccount.ID != 0 {
		return rhAccount.ID, nil
	}
	err = database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error
	if err != nil {
		utils.Log("err", err, "name", account).Warn("Error creating account")
	}
	return rhAccount.ID, err
}

func findAccount(c *gin.Context, identity *identity.Identity) bool {
	AccountIDCache.Lock.Lock()
	defer AccountIDCache.Lock.Unlock()

	if id, has := AccountIDCache.Values[identity.AccountNumber]; has {
		c.Set(KeyAccount, id)
	} else {
		// create new account if it does not exist
		accID, err := GetOrCreateAccount(identity.AccountNumber)
		if err != nil {
			return false
		}
		AccountIDCache.Values[identity.AccountNumber] = accID
		c.Set(KeyAccount, accID)
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
