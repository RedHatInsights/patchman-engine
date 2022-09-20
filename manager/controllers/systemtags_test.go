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
