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
	nUngrouped int64 = 9
	nAll       int64 = 18
)
var nonExisting = "00000000-0000-0000-3333-000000000000"

var testCases = []map[int64][]string{
	{nGroup1: {"00000000-0000-0000-0000-000000000001"}},
	{nGroup2: {"00000000-0000-0000-0000-000000000002"}},
	{nGroup1 + nGroup2: {"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"}},
	{nGroup1 + nUngrouped: {"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-999999999999"}},
	{nUngrouped: {nonExisting, "00000000-0000-0000-0000-999999999999"}},
	{0: {nonExisting}},
	{nUngrouped: {"00000000-0000-0000-0000-999999999999"}},
	{nAll: {"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-999999999999"}},
	{nAll: nil},
	{nAll: {}},
}

func TestApplyInventoryWorkspaceFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	for _, tc := range testCases {
		for expectedCount, workspaceIDs := range tc {
			var count int64
			ApplyInventoryWorkspaceFilter(DB.Table("system_inventory si").
				Joins("JOIN system_patch spatch ON si.id = spatch.system_id AND si.rh_account_id = spatch.rh_account_id"),
				workspaceIDs).Count(&count)
			assert.Equal(t, expectedCount, count)
		}
	}
}
