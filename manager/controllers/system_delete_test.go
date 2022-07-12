package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const del = "99c0ffee-0000-0000-0000-000000000de1"

func TestInitDelete(t *testing.T) {
	utils.TestLoadEnv("conf/test.env")
	SetupTest(t)

	assert.NoError(t, database.Db.Create(&models.SystemPlatform{
		InventoryID: del,
		RhAccountID: 1,
		DisplayName: del,
	}).Error)
	utils.TestLoadEnv("conf/manager.env")
}

func TestSystemDelete(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("DELETE", "/"+del, nil, nil, SystemDeleteHandler, 1, "DELETE", "/:inventory_id")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSystemDeleteWrongAccount(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("DELETE", "/"+del, nil, nil, SystemDeleteHandler, 2, "DELETE", "/:inventory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemDeleteNotFound(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("DELETE", "/"+del, nil, nil, SystemDeleteHandler, 1, "DELETE", "/:inventory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSystemDeleteUnknown(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithParams("DELETE", "/unknownsystem", nil, nil, SystemDeleteHandler, 1,
		"DELETE", "/:inventory_id")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
