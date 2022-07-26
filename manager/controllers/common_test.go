package controllers

import (
	"app/base/core"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Content type variables
var contentTypeCSV = "text/csv"
var contentTypeJSON = "application/json"

// Init request
func PrepareRequest(method string, url string, body io.Reader) (w *httptest.ResponseRecorder, req *http.Request) {
	req, _ = http.NewRequest(method, url, body)
	return httptest.NewRecorder(), req
}

// Check if additional header
func CheckHeader(req *http.Request, contentType *string) {
	if contentType != nil {
		contentTypeVal := *contentType
		req.Header.Add("Accept", contentTypeVal)
	}
}

// Create simple request
func CreateRequest(method string, url string, body io.Reader, contentType *string, handler gin.HandlerFunc) (
	w *httptest.ResponseRecorder) {
	w, req := PrepareRequest(method, url, body)
	CheckHeader(req, contentType)
	core.InitRouter(handler).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with params
func CreateRequestRouterWithParams(method string, url string, body io.Reader, contentType *string,
	handler gin.HandlerFunc, routerAccount int, routerMethod string, routerPath string) (w *httptest.ResponseRecorder) {
	w, req := PrepareRequest(method, url, body)
	CheckHeader(req, contentType)
	core.InitRouterWithParams(handler, routerAccount, routerMethod, routerPath).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with path
func CreateRequestRouterWithPath(method string, url string, body io.Reader, contentType *string,
	handler gin.HandlerFunc, routerPath string) (w *httptest.ResponseRecorder) {
	w, req := PrepareRequest(method, url, body)
	CheckHeader(req, contentType)
	core.InitRouterWithPath(handler, routerPath).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with account
func CreateRequestRouterWithAccount(method string, url string, body io.Reader, contentType *string,
	handler gin.HandlerFunc, routerPath string, routerAccount int) (w *httptest.ResponseRecorder) {
	w, req := PrepareRequest(method, url, body)
	CheckHeader(req, contentType)
	core.InitRouterWithAccount(handler, routerPath, routerAccount).ServeHTTP(w, req)
	return w
}

// Check status and parse response body
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, output interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
}
