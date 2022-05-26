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
	identity := utils.Identity{
		AccountNumber: "1234567",
		OrgID:         "9876543",
	}
	// account does not exist
	assert.NotContains(t, AccountIDCache.Values, identity.OrgID)
	// account does not exist but it is created
	firstFind := findAccount(c, &identity)
	assert.True(t, firstFind)
	assert.Contains(t, AccountIDCache.Values, identity.OrgID)
	accID1 := AccountIDCache.Values[identity.OrgID]
	// account exists
	secondFind := findAccount(c, &identity)
	assert.True(t, secondFind)
	assert.Contains(t, AccountIDCache.Values, identity.OrgID)
	accID2 := AccountIDCache.Values[identity.OrgID]
	// rhAccount.ID should be the same
	// second find should not create another record
	assert.Equal(t, accID1, accID2)
}

func TestUpdateOrgID(t *testing.T) {
	c := testSetup(t)
	accNrIdentity := utils.Identity{
		AccountNumber: "1111111",
	}
	// create account without OrgID
	accID, _ := GetOrCreateAccount(&accNrIdentity)
	AccountIDCache.Values[accNrIdentity.AccountNumber] = accID

	orgIDIdentity := utils.Identity{
		AccountNumber: "1111111",
		OrgID:         "7654321",
	}
	found := findAccount(c, &orgIDIdentity)
	assert.True(t, found)
	updatedID := AccountIDCache.Values[orgIDIdentity.OrgID]
	assert.Equal(t, updatedID, accID)
}

func TestFindDNoAccNr(t *testing.T) {
	c := testSetup(t)
	ident := utils.Identity{
		OrgID: "2222222",
	}
	// create account
	found := findAccount(c, &ident)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, ident.OrgID)
	accID := AccountIDCache.Values[ident.OrgID]
	// account exists
	found = findAccount(c, &ident)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, ident.OrgID)
	updatedID := AccountIDCache.Values[ident.OrgID]
	assert.Equal(t, updatedID, accID)
	// new account without AccountNumber
	// test that account `accID` without AccountNumber is not overwritten
	ident = utils.Identity{
		OrgID: "3333333",
	}
	found = findAccount(c, &ident)
	assert.True(t, found)
	assert.Contains(t, AccountIDCache.Values, ident.OrgID)
	newID := AccountIDCache.Values[ident.OrgID]
	assert.NotEqual(t, newID, accID)
}
