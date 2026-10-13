package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var subscriptionUUID = "cccccccc-0000-0000-0001-000000000004"
var orgID = "org_1"

const hostSubscriptionManagerUUID = "dddddddd-0000-0000-0001-000000000004"

func TestSubscribedSystemID(t *testing.T) {
	core.SetupTest(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(utils.KeyAccount, templateAccount)
	c.Set(utils.KeySystem, subscriptionUUID)
	c.Set(utils.KeyOrgID, orgID)
	account, org, systemID, err := getSubscribedSystem(c, database.DB)

	assert.Nil(t, err)
	assert.Equal(t, templateAccount, account)
	assert.Equal(t, orgID, org)
	assert.Equal(t, testInventoryID4, systemID)
}

func TestUnknownSubscribedSystemID(t *testing.T) {
	core.SetupTest(t)
	for _, tc := range []struct {
		cn      string
		account int
	}{
		{"cccccccc-0000-0000-0001-000000000001", templateAccount},
		{hostSubscriptionManagerUUID, templateAccount},
		{subscriptionUUID, templateAccount + 1},
	} {
		t.Run(tc.cn, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(utils.KeyAccount, tc.account)
			c.Set(utils.KeyOrgID, orgID)
			c.Set(utils.KeySystem, tc.cn)
			account, org, systemID, err := getSubscribedSystem(c, database.DB)
			assert.EqualError(t, err, "System "+tc.cn+" not found")
			assert.Equal(t, 0, account)
			assert.Equal(t, "", org)
			assert.Equal(t, uuid.Nil, systemID)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestUpdateTemplateSubscribedSystems(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, nil)

	w := CreateRequestRouterWithParams("PATCH", "/:template_id/subscribed-systems", templateUUID, "", nil, "",
		TemplateSubscribedSystemsUpdateHandler, templateAccount,
		core.ContextKV{Key: utils.KeySystem, Value: subscriptionUUID},
		core.ContextKV{Key: utils.KeyOrgID, Value: orgID})

	assert.Equal(t, http.StatusOK, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []uuid.UUID{testInventoryID4})
	database.DeleteTemplate(t, templateAccount, templateUUID)
}

func TestUpdateTemplateSubscribedSystemsInvalid(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, nil)

	w := CreateRequestRouterWithParams("PATCH", "/:template_id/subscribed-systems", templateUUID, "", nil, "",
		TemplateSubscribedSystemsUpdateHandler, templateAccount,
		core.ContextKV{Key: utils.KeySystem, Value: hostSubscriptionManagerUUID})

	assert.Equal(t, http.StatusNotFound, w.Code)
	database.CheckTemplateSystems(t, templateAccount, templateUUID, []uuid.UUID{})
	database.DeleteTemplate(t, templateAccount, templateUUID)
}
