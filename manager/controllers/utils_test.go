package controllers

import (
	"app/base/database"
	"app/base/rbac"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNestedQueryParse(t *testing.T) {
	q1 := map[string][]string{
		"filter[abc][efg][eq]": {"a"},
		"filter[a][]":          {"b", "c"},
		// Check that we stop after we encountered invalid filter syntax
		"filter[]]]]]": {},
		"filter[[[[[":  {},
		"filter":       {},
	}
	res := nestedQueryImpl(q1, "filter")
	res.Visit(func(keys []string, val string) {
		if reflect.DeepEqual([]string{"abc", "efg", "eq"}, keys) {
			assert.Equal(t, "a", val)
		}
		// We need to be able to parse multi-value elems
		if reflect.DeepEqual([]string{"a"}, keys) {
			assert.Contains(t, []string{"b", "c"}, val)
		}
	})
}

func TestNestedQueryInvalidKey(t *testing.T) {
	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		q := map[string][]string{
			"filter[abc][efg][eq]": {"a"},
		}
		res := nestedQueryImpl(q, "filte")
		assert.Equal(t, res, QueryMap{})
		done <- true
	}()

	select {
	case <-timeout:
		t.Fatal("Timeout exceeded - probably infinite loop in nested query")
	case <-done:
	}
}

func TestGroupNameFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	database.Configure()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("GET", "/?filter[group_name][in]=group2", nil)

	filters, err := ParseInventoryFilters(c)
	assert.Nil(t, err)

	var systems []SystemsID
	groups := map[string]string{
		rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]","[{\"id\":\"inventory-group-2\"}]"}`,
	}
	tx := database.Systems(database.Db, 1, groups)
	tx, _ = ApplyInventoryFilter(filters, tx, "sp.inventory_id")
	tx.Scan(&systems)

	assert.Equal(t, 2, len(systems)) // 2 systems with `group2` in test_data
	assert.Equal(t, "00000000-0000-0000-0000-000000000007", systems[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000008", systems[1].ID)
}
