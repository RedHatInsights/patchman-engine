package middlewares

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const KeyAccount = "account"

var AccountIDCache = struct {
	Values map[string]int
	Lock   sync.Mutex
}{Values: map[string]int{}, Lock: sync.Mutex{}}

// Stores or updates the account data, returning the account id
func GetOrCreateAccount(identity *utils.Identity) (int, error) {
	rhAccount := models.RhAccount{
		Name:  identity.GetAccountNumber(),
		OrgID: &identity.OrgID,
	}
	var err error
	if rhAccount.OrgID == nil || *rhAccount.OrgID == "" {
		// Identity from kafka msg, missing OrgID in msg from Inventory, find by AccountNumber
		// https://issues.redhat.com/browse/ESSNTL-2525
		err = database.Db.Where("name = ?", *rhAccount.Name).Find(&rhAccount).Error
		if err != nil {
			utils.Log("err", err, "name", *rhAccount.Name).Warn("Error in finding account")
		}
	} else {
		// Find account by OrgID
		err = database.Db.Where("org_id = ?", *rhAccount.OrgID).Find(&rhAccount).Error
		if err != nil {
			utils.Log("err", err, "org_id", *rhAccount.OrgID).Warn("Error in finding account")
		}
	}
	if rhAccount.ID != 0 {
		return rhAccount.ID, nil
	}

	// OrgID not found, try to find account by account_number and update org_id or create new record
	if rhAccount.OrgID == nil || *rhAccount.OrgID == "" {
		// kafka msg without OrgID, create account by AccountNumber
		err = database.OnConflictUpdate(database.Db, "name", "name").Select("name").Create(&rhAccount).Error
		if err != nil {
			utils.Log("err", err, "name", *rhAccount.Name).Warn("Error creating account")
		}
		return rhAccount.ID, err
	}
	// create new rhAccount with OrgID
	if rhAccount.Name == nil || *rhAccount.Name == "" {
		// avoid updating other account with name=null if AccountNumber=""
		err = database.OnConflictUpdate(database.Db, "org_id", "org_id").Select("org_id").Create(&rhAccount).Error
		if err != nil {
			utils.Log("err", err, "org_id", *rhAccount.OrgID).Warn("Error creating account")
		}
		return rhAccount.ID, err
	}
	// create/update with OrgID
	err = database.OnConflictUpdate(database.Db, "name", "org_id").Create(&rhAccount).Error
	if err != nil {
		utils.Log("err", err, "org_id", *rhAccount.OrgID).Warn("Error updating org_id/creating account")
	}
	return rhAccount.ID, err
}

func findAccount(c *gin.Context, identity *utils.Identity) bool {
	AccountIDCache.Lock.Lock()
	defer AccountIDCache.Lock.Unlock()

	if id, has := AccountIDCache.Values[identity.OrgID]; has {
		c.Set(KeyAccount, id)
	} else {
		// create new account if it does not exist
		accID, err := GetOrCreateAccount(identity)
		if err != nil {
			return false
		}
		AccountIDCache.Values[identity.OrgID] = accID
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
