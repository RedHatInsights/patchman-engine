package utils

import (
	"app/base/types/vmaas"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compareUpdatesMerge(t *testing.T, jsonA, jsonB, merged []byte) {
	var listA, listB vmaas.UpdatesV2ResponseUpdateList
	err := json.Unmarshal(jsonA, &listA)
	assert.Nil(t, err)
	err = json.Unmarshal(jsonB, &listB)
	assert.Nil(t, err)

	listMerged, err := mergeUpdates(listA, listB)
	assert.Nil(t, err)

	res, err := json.Marshal(listMerged)
	assert.Nil(t, err)

	require.JSONEq(t, string(merged), string(res))
}

func TestMergeEmptyStruct(t *testing.T) {
	compareUpdatesMerge(t, emptyStruct, pkgA1, pkgA1)
	compareUpdatesMerge(t, pkgA1, emptyStruct, pkgA1)
}

func TestMergeEmptyAvailableUpdates(t *testing.T) {
	compareUpdatesMerge(t, emptyUpdate, pkgA1, pkgA1)
	compareUpdatesMerge(t, pkgA1, emptyUpdate, pkgA1)
}

func TestMergeTwoDifferent(t *testing.T) {
	compareUpdatesMerge(t, pkgA1, pkgA2, pkgA12)
	compareUpdatesMerge(t, pkgA2, pkgA1, pkgA12)
}

func TestMergeDuplicate(t *testing.T) {
	compareUpdatesMerge(t, pkgA2, pkgA123, pkgA123)
	compareUpdatesMerge(t, pkgA123, pkgA2, pkgA123)
}

func TestMergeDifferentAttrs(t *testing.T) {
	compareUpdatesMerge(t, pkgA1Xattrs, pkgA123, pkgA123Xattrs)
	compareUpdatesMerge(t, pkgA123, pkgA1Xattrs, pkgA123Xattrs)
}

func compareResponseMerge(t *testing.T, jsonA, jsonB, merged []byte) {
	var respA, respB vmaas.UpdatesV2Response

	err := json.Unmarshal(jsonA, &respA)
	assert.Nil(t, err)
	err = json.Unmarshal(jsonB, &respB)
	assert.Nil(t, err)

	respMerged, err := MergeVMaaSResponses(&respA, &respB)
	assert.Nil(t, err)

	res, err := json.Marshal(respMerged)
	assert.Nil(t, err)

	require.JSONEq(t, string(merged), string(res))
}

func TestMergeVMaaSResponses(t *testing.T) {
	compareResponseMerge(t, kernel3101, kernel3102, kernel3101and3102)
	compareResponseMerge(t, kernel3102, kernel3101, kernel3101and3102)
}

func TestMergeVMaaSResponses2(t *testing.T) {
	// keep only the latest package
	// if a customer has 2 kernel versions installed
	// we should display latest updates
	compareResponseMerge(t, kernel3111, kernel3121, kernel3111AndKernel3121)
	compareResponseMerge(t, kernel3121, kernel3111, kernel3111AndKernel3121)
}

func TestMergeVMaaSResponses3(t *testing.T) {
	compareResponseMerge(t, bash44201, bash44202, bash44201AndBash44202)
	compareResponseMerge(t, bash44202, bash44201, bash44201AndBash44202)
}
