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

// Add content type to header
func acceptContentType(req *http.Request, ct string) {
	if ct != "" {
		req.Header.Add("Accept", ct)
	}
}

// Init request
func prepareRequest(method string, url string, body io.Reader, ct string) (w *httptest.ResponseRecorder,
	req *http.Request) {
	req, _ = http.NewRequest(method, url, body)
	acceptContentType(req, ct)
	return httptest.NewRecorder(), req
}

// Create simple request
func CreateRequest(method string, url string, body io.Reader, contentType string, handler gin.HandlerFunc) (
	w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	core.InitRouter(handler).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with params
func CreateRequestRouterWithParams(method string, url string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerAccount int, routerMethod string, routerPath string) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	core.InitRouterWithParams(handler, routerAccount, routerMethod, routerPath).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with path
func CreateRequestRouterWithPath(method string, url string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerPath string) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	core.InitRouterWithPath(handler, routerPath).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with account
func CreateRequestRouterWithAccount(method string, url string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerPath string, routerAccount int) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	core.InitRouterWithAccount(handler, routerPath, routerAccount).ServeHTTP(w, req)
	return w
}

// Check status and parse response body
func CheckResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, output interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
}
