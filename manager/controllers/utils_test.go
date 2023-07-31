package controllers

import (
	"app/base/database"
	"app/base/rbac"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGroupNameFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/?filter[group_name]=group2", nil)

	filters, err := ParseAllFilters(c, ListOpts{})
	assert.Nil(t, err)

	var systems []SystemsID
	groups := map[string]string{
		rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]","[{\"id\":\"inventory-group-2\"}]"}`,
	}
	tx := database.Systems(database.Db, 1, groups)
	tx, _ = ApplyInventoryFilter(inventoryFilters, tx, "sp.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 2, len(systems)) // 2 systems with `group2` in test_data
	assert.Equal(t, "00000000-0000-0000-0000-000000000007", systems[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000008", systems[1].ID)
}

func TestGroupNameFilter2(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/?filter[group_name]=group1,group2", nil)

	filters, err := ParseAllFilters(c, ListOpts{})
	assert.Nil(t, err)

	var systems []SystemsID
	groups := map[string]string{
		rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]","[{\"id\":\"inventory-group-2\"}]"}`,
	}
	tx := database.Systems(database.Db, 1, groups)
	tx, _ = ApplyInventoryFilter(inventoryFilters, tx, "sp.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 8, len(systems)) // 2 systems with `group2`, 6 with `group1` in test_data
}
