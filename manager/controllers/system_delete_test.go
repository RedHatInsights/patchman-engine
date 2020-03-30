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

func TestInitDelete(t *testing.T) {
	utils.TestLoadEnv("conf/test.env")
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: "DEL-1",
		RhAccountID: 1,
		DisplayName: "DEL-1",
	}).Error)
	utils.TestLoadEnv("conf/manager.env")
	core.SetupTestEnvironment()
}

func TestSystemDelete(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	core.InitRouterWithParams(SystemDeleteHandler, "1", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSystemDeleteWrongAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	core.InitRouterWithParams(SystemDeleteHandler, "2", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestSystemDeleteNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/DEL-1", nil)
	core.InitRouterWithParams(SystemDeleteHandler, "1", "DELETE", "/:inventory_id").ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}
