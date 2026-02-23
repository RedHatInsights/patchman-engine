package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var subscriptionUUID = "cccccccc-0000-0000-0001-000000000004"
var templateSystemUUID = "00000000-0000-0000-0000-000000000004"
var subscriptionInvalidUUID = "99999999-9999-8888-8888-888888888888"
var orgID = "org_1"

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
	c.Set(utils.KeySystem, "cccccccc-0000-0000-0001-000000000001")
	account, systemID, err := getSubscribedSystem(c, database.DB)

	assert.EqualError(t, err, "System cccccccc-0000-0000-0001-000000000001 not found")
	assert.Equal(t, 0, account)
	assert.Equal(t, "", systemID)
}

func TestUpdateTemplateSubscribedSystems(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, nil)

	w := CreateRequestRouterWithParams("PATCH", "/:template_id/subscribed-systems", templateUUID, "", nil, "",
		TemplateSubscribedSystemsUpdateHandler, templateAccount,
		core.ContextKV{Key: utils.KeySystem, Value: subscriptionUUID},
		core.ContextKV{Key: utils.KeyOrgID, Value: orgID})

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{templateSystemUUID})
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateSubscribedSystemsInvalid(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, nil)

	w := CreateRequestRouterWithParams("PATCH", "/:template_id/subscribed-systems", templateUUID, "", nil, "",
		TemplateSubscribedSystemsUpdateHandler, templateAccount,
		core.ContextKV{Key: utils.KeySystem, Value: subscriptionInvalidUUID})

	assert.Equal(t, http.StatusNotFound, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{})
	database.DeleteTemplate(t, templateAccount, templateUUID)
}
