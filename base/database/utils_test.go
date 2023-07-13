package database

import (
	"app/base/rbac"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	// counts of systems from system_platform JOIN inventory.hosts
	nGroup1    int64 = 6
	nGroup2    int64 = 2
	nUngrouped int64 = 6
	nAll       int64 = 16
)

// nolint: lll
var testCases = []map[int64]map[string]string{
	{nGroup1: {rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]"}`}},
	{nGroup2: {rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-2\"}]"}`}},
	{nGroup1 + nGroup2: {rbac.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]","[{\"id\":\"inventory-group-2\"}]"}`}},
	{nGroup1 + nUngrouped: {
		rbac.KeyGrouped:   `{"[{\"id\":\"inventory-group-1\"}]"}`,
		rbac.KeyUngrouped: "[]",
	}},
	{nUngrouped: {
		rbac.KeyGrouped:   `{"[{\"id\":\"non-existing-group\"}]"}`,
		rbac.KeyUngrouped: "[]",
	}},
	{0: {rbac.KeyGrouped: `{"[{\"id\":\"non-existing-group\"}]"}`}},
	{nAll: {}},
}

func TestInventoryHostsJoin(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	for _, tc := range testCases {
		for expectedCount, groups := range tc {
			var count int64
			InventoryHostsJoin(Db.Table("system_platform sp"), groups).Count(&count)
			assert.Equal(t, expectedCount, count)
		}
	}
}
