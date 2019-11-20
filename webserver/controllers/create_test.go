package controllers

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/base/core"
	"app/base/database"
	"app/base/structures"
)

// test create record
func TestCreate1(t *testing.T) {
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?id=12&value=1.23", nil)
	initRouter(CreateHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var records []structures.HostDAO
	err := database.Db.Model(&structures.HostDAO{}).Find(&records).Error
	assert.Nil(t, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, 12, records[0].ID)
	assert.Equal(t, "req", records[0].Request)
	assert.Equal(t, "chs", records[0].Checksum)
}
