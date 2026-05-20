package database

import (
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	// counts of systems from system_inventory (+ system_patch join in Systems())
	nGroup1    int64 = 7
	nGroup2    int64 = 2
	nUngrouped int64 = 7
	nAll       int64 = 18
)

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

func TestApplyInventoryWorkspaceFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	for _, tc := range testCases {
		for expectedCount, groups := range tc {
			var count int64
			ApplyInventoryWorkspaceFilter(DB.Table("system_inventory si").
				Joins("JOIN system_patch spatch ON si.id = spatch.system_id AND si.rh_account_id = spatch.rh_account_id"),
				groups).Count(&count)
			assert.Equal(t, expectedCount, count)
		}
	}
}
