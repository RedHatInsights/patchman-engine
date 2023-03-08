package controllers

import (
	"app/base/core"
	"app/manager/middlewares"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testAccountSystemCounts(t *testing.T, acc int, count int) {
	core.SetupTest(t)
	var output SystemsResponseV3
	w := CreateRequestRouterWithAccount("GET", "/", nil, "", SystemsListHandler, "/", acc,
		core.ContextKV{Key: middlewares.KeyApiver, Value: 3})
	CheckResponse(t, w, http.StatusOK, &output)
	// data
	assert.Equal(t, count, len(output.Data))
}

func TestMissingAccount(t *testing.T) {
	testAccountSystemCounts(t, 0, 0)
	testAccountSystemCounts(t, 1, 8)
	testAccountSystemCounts(t, 2, 3)
	testAccountSystemCounts(t, 3, 4)
	testAccountSystemCounts(t, 4, 0)
}
