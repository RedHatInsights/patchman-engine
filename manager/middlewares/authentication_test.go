package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func testSetup(t *testing.T) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	utils.SkipWithoutDB(t)
	utils.SetDefaultEnvOrFail("LOG_LEVEL", "debug")
	utils.ConfigureLogging()
	database.Configure()
	return c
}

func TestFindAccount(t *testing.T) {
	c := testSetup(t)
	orgID := "9876543"
	// account does not exist
	assert.NotContains(t, AccountIDCache.Values, orgID)
	// account does not exist but it is created
	firstFind := findAccount(c, orgID)
	assert.True(t, firstFind)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID1 := AccountIDCache.Values[orgID]
	// account exists
	secondFind := findAccount(c, orgID)
	assert.True(t, secondFind)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID2 := AccountIDCache.Values[orgID]
	// rhAccount.ID should be the same
	// second find should not create another record
	assert.Equal(t, accID1, accID2)
}

func TestUpdateOrgID(t *testing.T) {
	c := testSetup(t)
	orgID := "7654321"
	found := findAccount(c, orgID)
	assert.True(t, found)
	updatedID := AccountIDCache.Values[orgID]
	assert.NotEqual(t, updatedID, 0)
}

func TestFindDNoAccNr(t *testing.T) {
	c := testSetup(t)
	orgID := "2222222"
	// create account
	found := findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	accID := AccountIDCache.Values[orgID]
	// account exists
	found = findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	updatedID := AccountIDCache.Values[orgID]
	assert.Equal(t, updatedID, accID)
	// new account without AccountNumber
	// test that account `accID` without AccountNumber is not overwritten
	orgID = "3333333"
	found = findAccount(c, orgID)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, orgID)
	newID := AccountIDCache.Values[orgID]
	assert.NotEqual(t, newID, accID)
}
