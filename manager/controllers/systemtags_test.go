package controllers

import (
	"net/http"
	"testing"

	"app/base/core"

	"github.com/stretchr/testify/assert"
)

// test System Tags List
func TestSystemTagsListDefault(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", "", "", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)
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

	w := CreateRequestRouterWithAccount("GET", "/", "", "?offset=1&limit=1", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	if !assert.Equal(t, 1, len(output.Data)) {
		return
	}

	assert.Equal(t, 2, output.Data[0].Count)
	assert.Equal(t, "k2", output.Data[0].Tag.Key)
	assert.Equal(t, "ns1", output.Data[0].Tag.Namespace)
	assert.Equal(t, "val2", output.Data[0].Tag.Value)
}

func TestSystemTagsTotalItems(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", "", "", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 4, output.Meta.TotalItems)
}

func TestSystemTagsListSort(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", "", "?sort=count", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)
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

	w := CreateRequestRouterWithAccount("GET", "/", "", "?sort=id", nil, "", SystemTagListHandler, 1)
	assert.Equal(t, 400, w.Code)
}

func TestSystemTagsListSearch(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", "", "?search=k3", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 2, output.Meta.TotalItems)
	assert.Equal(t, "k3", output.Meta.Search)

	assert.Equal(t, 1, output.Data[0].Count)
	assert.Equal(t, "k3", output.Data[0].Tag.Key)
	assert.Equal(t, "ns1", output.Data[0].Tag.Namespace)
	assert.Equal(t, "val3", output.Data[0].Tag.Value)

	assert.Equal(t, 3, output.Data[1].Count)
	assert.Equal(t, "k3", output.Data[1].Tag.Key)
	assert.Equal(t, "ns1", output.Data[1].Tag.Namespace)
	assert.Equal(t, "val4", output.Data[1].Tag.Value)
}

func TestSystemTagsListSearchUnknown(t *testing.T) {
	core.SetupTest(t)

	w := CreateRequestRouterWithAccount("GET", "/", "", "?search=unknown", nil, "", SystemTagListHandler, 1)

	var output SystemTagsResponse
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 0, len(output.Data))
	assert.Equal(t, "unknown", output.Meta.Search)
}
