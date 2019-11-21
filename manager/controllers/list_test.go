package controllers

import (
	"app/base/core"
	"github.com/bitly/go-simplejson"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)


func TestList(t *testing.T) {
	core.SetupTestEnvironment()

	createTestingSample(1)
	createTestingSample(2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	initRouter(ListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	js, err := simplejson.NewJson([]byte(w.Body.String()))
	assert.Nil(t, err)

	pkgs, err := js.Array()
	assert.Equal(t, 2, len(pkgs))
}

func TestGetHostOK(t *testing.T) {
	core.SetupTestEnvironment()

	createTestingSample(1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/1", nil)
	initRouterWithPath(GetHostHandler, "/:id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"id":1,"request":"r","checksum":"454349e422f05297191ead13e21d3db520e5abef52055e4964b82fb213f593a1","updated":"0001-01-01T00:00:00Z"}`, w.Body.String())
}

func TestGetHostNotFound(t *testing.T) {
	core.SetupTestEnvironment()

	createTestingSample(1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/100", nil)
	initRouterWithPath(GetHostHandler, "/:id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, `{"err":"record not found"}`, w.Body.String())
}
