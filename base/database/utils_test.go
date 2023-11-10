package database

import (
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
	{nGroup1: {utils.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]"}`}},
	{nGroup2: {utils.KeyGrouped: `{"[{\"id\":\"inventory-group-2\"}]"}`}},
	{nGroup1 + nGroup2: {utils.KeyGrouped: `{"[{\"id\":\"inventory-group-1\"}]","[{\"id\":\"inventory-group-2\"}]"}`}},
	{nGroup1 + nUngrouped: {
		utils.KeyGrouped:   `{"[{\"id\":\"inventory-group-1\"}]"}`,
		utils.KeyUngrouped: "[]",
	}},
	{nUngrouped: {
		utils.KeyGrouped:   `{"[{\"id\":\"non-existing-group\"}]"}`,
		utils.KeyUngrouped: "[]",
	}},
	{0: {utils.KeyGrouped: `{"[{\"id\":\"non-existing-group\"}]"}`}},
	{nUngrouped: {utils.KeyUngrouped: "[]"}},
	{nAll: {}},
	{nAll: nil},
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
