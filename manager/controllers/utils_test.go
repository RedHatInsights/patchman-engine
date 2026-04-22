package controllers

import (
	"app/base/database"
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
	workspaceIDs := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}
	tx := database.Systems(database.DB, 1, workspaceIDs)
	tx, _ = ApplyInventoryFilter(filters, tx, "si.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 2, len(systems))
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
	workspaceIDs := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}
	tx := database.Systems(database.DB, 1, workspaceIDs)
	tx, _ = ApplyInventoryFilter(filters, tx, "si.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 9, len(systems))
}
