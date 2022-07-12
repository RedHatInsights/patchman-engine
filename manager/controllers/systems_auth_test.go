package controllers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testAccountSystemCounts(t *testing.T, acc int, count int) {
	SetupTest(t)
	var output SystemsResponse
	w := CreateRequestRouterWithAccount("GET", "/", nil, nil, SystemsListHandler, "/", acc)

	assert.Equal(t, http.StatusOK, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
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
