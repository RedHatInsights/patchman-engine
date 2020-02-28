package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInit(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: "DEL-1",
		RhAccountID: 0,
	}).Error)
}

func TestSystemDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	initRouterWithParams(SystemDeleteHandler, "0", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSystemDeleteWrongAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	initRouterWithParams(SystemDeleteHandler, "1", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestSystemDeleteNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	initRouterWithParams(SystemDeleteHandler, "0", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}
