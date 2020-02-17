package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/INV-0", nil)
	initRouterWithPath(SystemDeleteHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSystemDeleteNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/INV-0", nil)
	initRouterWithPath(SystemDeleteHandler, "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}
