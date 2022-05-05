package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/stretchr/testify/assert"
)

func testSetup() {
	utils.SetDefaultEnvOrFail("LOG_LEVEL", "debug")
	utils.ConfigureLogging()
	database.Configure()
}

func TestFindAccount(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	identity := identity.Identity{
		AccountNumber: "1234567",
	}
	utils.SkipWithoutDB(t)
	testSetup()
	// account does not exist
	assert.NotContains(t, AccountIDCache.Values, identity.AccountNumber)
	// account does not exist but it is created
	firstFind := findAccount(c, &identity)
	assert.True(t, firstFind)
	assert.Contains(t, AccountIDCache.Values, identity.AccountNumber)
	// account exists
	secondFind := findAccount(c, &identity)
	assert.True(t, secondFind)
	assert.Contains(t, AccountIDCache.Values, identity.AccountNumber)
}
