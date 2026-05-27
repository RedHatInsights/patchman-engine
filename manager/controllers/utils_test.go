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
	"gorm.io/gorm"
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
		utils.KeyGrouped: `{"[{\"id\":\"aaaaaaaa-0000-0000-0000-000000000001\"}]","[{\"id\":\"aaaaaaaa-0000-0000-0000-000000000002\"}]"}`, //nolint:lll
	}
	tx := database.Systems(database.DB, 1, groups)
	tx, _ = ApplyInventoryFilter(filters, tx, "si.inventory_id")
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
		utils.KeyGrouped: `{"[{\"id\":\"aaaaaaaa-0000-0000-0000-000000000001\"}]","[{\"id\":\"aaaaaaaa-0000-0000-0000-000000000002\"}]"}`, //nolint:lll
	}
	tx := database.Systems(database.DB, 1, groups)
	tx, _ = ApplyInventoryFilter(filters, tx, "si.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 9, len(systems)) // 2 systems with `group2`, 6 with `group1` in test_data
}

func TestApplySearchEmpty(t *testing.T) {
	var baseTx gorm.DB

	var cases = []struct {
		url     string
		columns []string
		comment string
	}{
		{"/", []string{""}, "no search query returns original tx and empty meta"},
		{"/?search=", []string{"some_col"}, "empty search value returns original tx and empty meta"},
		{"/?search=foo", []string{}, "search present but no columns returns original tx and empty meta"},
	}
	for _, c := range cases {
		t.Run(c.comment, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request, _ = http.NewRequest("GET", c.url, nil)
			outTx, q := ApplySearch(ctx, &baseTx, c.columns...)
			assert.Same(t, &baseTx, outTx)
			assert.Equal(t, "", q)
		})
	}
}

func TestApplySearchHello(t *testing.T) {
	core.SetupTest(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/?search=hello", nil)

	tx, searchQ := ApplySearch(c, database.DB, "a", "b")
	assert.Equal(t, "search=hello", searchQ)

	tx = tx.Where("1 = 0")
	var n string
	sql := tx.ToSQL(func(g *gorm.DB) *gorm.DB {
		return g.Table("(select 'a' as a, 'b' as b) as t").Select("a").Scan(&n)
	})

	assert.Contains(t, sql, "WHERE (a::text ILIKE '%hello%' OR b::text ILIKE '%hello%')")
}
