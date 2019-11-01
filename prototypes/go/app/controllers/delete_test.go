package controllers

import (
	"gin-container/app/core"
	"gin-container/app/database"
	"gin-container/app/structures"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDelete1(t *testing.T) {
	core.SetupTestEnvironment()

	createTestingSample(1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?id=1", nil)
	initRouter(DeleteHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var records []structures.HostDAO
	err := database.Db.Model(&structures.HostDAO{}).Find(&records).Error
	assert.Nil(t, err)
	assert.Equal(t, 0, len(records))
}

func TestDelete2(t *testing.T) {
	core.SetupTestEnvironment()

	createTestingSample(1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?id=2", nil)
	initRouter(DeleteHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var records []structures.HostDAO
	err := database.Db.Model(&structures.HostDAO{}).Find(&records).Error
	assert.Nil(t, err)
	assert.Equal(t, 1, len(records))
}
