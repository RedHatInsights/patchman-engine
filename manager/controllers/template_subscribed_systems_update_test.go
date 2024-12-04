package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var subscriptionUUID = "cccccccc-0000-0000-0001-000000000004"
var templateSystemUUID = "00000000-0000-0000-0000-000000000004"

func TestSubscribedSystemID(t *testing.T) {
	core.SetupTest(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(utils.KeyAccount, templateAccount)
	c.Set(utils.KeySystem, subscriptionUUID)
	account, systemID, err := getSubscribedSystem(c, database.DB)

	assert.Nil(t, err)
	assert.Equal(t, templateAccount, account)
	assert.Equal(t, templateSystemUUID, systemID)
}

func TestUnknownSubscribedSystemID(t *testing.T) {
	core.SetupTest(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(utils.KeyAccount, templateAccount)
	c.Set(utils.KeySystem, "unknown-uuid")
	account, systemID, err := getSubscribedSystem(c, database.DB)

	assert.EqualError(t, err, "System unknown-uuid not found")
	assert.Equal(t, 0, account)
	assert.Equal(t, "", systemID)
}
