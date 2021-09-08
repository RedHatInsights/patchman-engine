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

const del = "99c0ffee-0000-0000-0000-000000000de1"

func TestInitDelete(t *testing.T) {
	utils.TestLoadEnv("conf/test.env")
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: del,
		RhAccountID: 1,
		DisplayName: del,
	}).Error)
	utils.TestLoadEnv("conf/manager.env")
}

func TestSystemDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/"+del, nil)
	core.InitRouterWithParams(SystemDeleteHandler, 1, "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSystemDeleteWrongAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/"+del, nil)
	core.InitRouterWithParams(SystemDeleteHandler, 2, "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemDeleteNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/"+del, nil)
	core.InitRouterWithParams(SystemDeleteHandler, 1, "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemDeleteUnknown(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/unknownsystem", nil)
	core.InitRouterWithParams(SystemDeleteHandler, 1, "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
