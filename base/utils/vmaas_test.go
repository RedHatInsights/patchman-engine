package utils

import (
	"app/base/vmaas"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestMergeUpdates(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash2, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 8, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithDuplicates(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash3, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash4, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 6, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithALonger(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash5, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash6, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 8, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithAShorter(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash6, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash5, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 8, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithALesserLonger(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash7, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash8, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 7, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithAGreaterLonger(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash9, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash10, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 7, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithALesserShorter(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash11, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash12, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 8, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithAGreaterShorter(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash8, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash7, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 7, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func TestMergeUpdatesWithDifferentRepos(t *testing.T) {
	var listA vmaas.UpdatesV2ResponseUpdateList
	var listB vmaas.UpdatesV2ResponseUpdateList

	err := json.Unmarshal(updatesBash13, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(updatesBash14, &listB)
	assert.Nil(t, err)

	merged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	assert.Equal(t, 10, len(merged.GetAvailableUpdates()))
	assert.Equal(t, true, updatesAscending(merged.GetAvailableUpdates()))
}

func updatesAscending(updates []vmaas.UpdatesV2ResponseAvailableUpdates) bool {
	var prev string
	var prevRepo string

	for i, u := range updates {
		nevra, err := ParseNevra(u.GetPackage())
		if err != nil {
			return false
		}

		if i == 0 {
			prev = nevra.Version
			prevRepo = *u.Repository
			continue
		}
		// Simplified, ideally should be map check.
		if prev >= nevra.Version && prevRepo == *u.Repository {
			return false
		}
	}

	return true
}

func TestMergeVMaaSResponses(t *testing.T) {
	var respA vmaas.UpdatesV2Response
	var respB vmaas.UpdatesV2Response

	err := json.Unmarshal(kernel3101, &respA)
	assert.Nil(t, err)
	err = json.Unmarshal(kernel3102, &respB)
	assert.Nil(t, err)

	res, err := MergeVMaaSResponses(&respA, &respB)
	assert.Nil(t, err)

	updateList := (*res.UpdateList)[kernel310Pkg]
	assert.Equal(t, 6, len(updateList.GetAvailableUpdates()))
}

func TestMergeVMaaSResponses2(t *testing.T) {
	var respA vmaas.UpdatesV2Response
	var respB vmaas.UpdatesV2Response
	var merged vmaas.UpdatesV2Response

	err := json.Unmarshal(kernel3111, &respA)
	assert.Nil(t, err)
	err = json.Unmarshal(kernel3121, &respB)
	assert.Nil(t, err)
	err = json.Unmarshal(kernel3111AndKernel3121, &merged)
	assert.Nil(t, err)

	res, err := MergeVMaaSResponses(&respA, &respB)
	assert.Nil(t, err)

	updateList := (*res.UpdateList)[kernel3111Pkg]
	updateList2 := (*res.UpdateList)[kernel3121Pkg]
	updateListMerged := (*merged.UpdateList)[kernel3111Pkg]
	updateListMerged2 := (*merged.UpdateList)[kernel3121Pkg]
	assert.Equal(t, len(updateListMerged.GetAvailableUpdates()), len(updateList.GetAvailableUpdates()))
	assert.Equal(t, len(updateListMerged2.GetAvailableUpdates()), len(updateList2.GetAvailableUpdates()))
	assert.Equal(t, len(merged.GetUpdateList()), len(res.GetUpdateList()))
}

func TestMergeVMaaSResponses3(t *testing.T) {
	var respA vmaas.UpdatesV2Response
	var respB vmaas.UpdatesV2Response
	var merged vmaas.UpdatesV2Response

	err := json.Unmarshal(bash44201, &respA)
	assert.Nil(t, err)
	err = json.Unmarshal(bash44202, &respB)
	assert.Nil(t, err)
	err = json.Unmarshal(bash44201AndBash44202, &merged)
	assert.Nil(t, err)

	res, err := MergeVMaaSResponses(&respA, &respB)
	assert.Nil(t, err)

	updateList := (*res.UpdateList)[bash4420Pkg]
	updateList2 := (*res.UpdateList)[bash4420Pkg]
	updateListMerged := (*merged.UpdateList)[bash4420Pkg]
	updateListMerged2 := (*merged.UpdateList)[bash4420Pkg]
	assert.Equal(t, len(updateListMerged.GetAvailableUpdates()), len(updateList.GetAvailableUpdates()))
	assert.Equal(t, len(updateListMerged2.GetAvailableUpdates()), len(updateList2.GetAvailableUpdates()))
	assert.Equal(t, len(merged.GetUpdateList()), len(res.GetUpdateList()))

	if !reflect.DeepEqual(res.GetUpdateList(), merged.GetUpdateList()) {
		t.Fatal("update lists are not equal\n")
	}
}
