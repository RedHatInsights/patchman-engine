package controllers

import (
	"app/base/core"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test System Tags List
func TestSystemTagsListDefault(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", nil, "", SystemTagListHandler, "/", 1)

	var output SystemTagsResponse
	CheckResponse(t, w, 200, &output)
	if !assert.Equal(t, 4, len(output.Data)) {
		return
	}

	assert.Equal(t, 7, output.Data[0].Count)
	assert.Equal(t, "k1", output.Data[0].Tag.Key)
	assert.Equal(t, "ns1", output.Data[0].Tag.Namespace)
	assert.Equal(t, "val1", output.Data[0].Tag.Value)
}

func TestSystemTagsListPagination(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/?offset=1&limit=1", nil, "", SystemTagListHandler, "/", 1)

	var output SystemTagsResponse
	CheckResponse(t, w, 200, &output)
	if !assert.Equal(t, 1, len(output.Data)) {
		return
	}

	assert.Equal(t, 2, output.Data[0].Count)
	assert.Equal(t, "k2", output.Data[0].Tag.Key)
	assert.Equal(t, "ns1", output.Data[0].Tag.Namespace)
	assert.Equal(t, "val2", output.Data[0].Tag.Value)
}

func TestSystemTagsListSort(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/?sort=count", nil, "", SystemTagListHandler, "/", 1)

	var output SystemTagsResponse
	CheckResponse(t, w, 200, &output)
	if !assert.Equal(t, 4, len(output.Data)) {
		return
	}

	assert.Equal(t, 1, output.Data[0].Count)
	assert.Equal(t, "k3", output.Data[0].Tag.Key)
	assert.Equal(t, "ns1", output.Data[0].Tag.Namespace)
	assert.Equal(t, "val3", output.Data[0].Tag.Value)
}

func TestSystemTagsListBadRequestOnIdtSort(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/?sort=id", nil, "", SystemTagListHandler, "/", 1)
	assert.Equal(t, 400, w.Code)
}
