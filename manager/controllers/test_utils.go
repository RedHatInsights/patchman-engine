package controllers

import (
	"app/base/core"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const InvalidContentTypeErr = `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`

func ParseResponseBody(t *testing.T, bytes []byte, out interface{}) {
	// don't use sonic.Unmarshal as some tests receive empty output
	err := json.Unmarshal(bytes, &out)
	assert.Nil(t, err, string(bytes))
}

// Add content type to header
func acceptContentType(req *http.Request, ct string) {
	if ct != "" {
		req.Header.Add("Accept", ct)
	}
}

// Fill params and append query string to routerPath to get url
func getURLFromRouterPath(routerPath, param, queryString string) string {
	pattern := regexp.MustCompile(":[a-zA-Z0-9_]+")
	return pattern.ReplaceAllString(routerPath, param) + queryString
}

// Init request
func prepareRequest(method string, url string, body io.Reader, ct string) (w *httptest.ResponseRecorder,
	req *http.Request) {
	req, _ = http.NewRequest(method, url, body)
	acceptContentType(req, ct)
	return httptest.NewRecorder(), req
}

// Create simple request
func CreateRequest(method string, url string, body io.Reader, contentType string, handler gin.HandlerFunc,
	contextKVs ...core.ContextKV) (
	w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	core.InitRouter(handler, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with params
func CreateRequestRouterWithParams(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerAccount int, contextKVs ...core.ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	core.InitRouterWithParams(handler, routerAccount, method, routerPath, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with path
func CreateRequestRouterWithPath(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, contextKVs ...core.ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	core.InitRouterWithPath(handler, routerPath, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with account
func CreateRequestRouterWithAccount(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerAccount int, contextKVs ...core.ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	core.InitRouterWithAccount(handler, routerPath, routerAccount, contextKVs...).ServeHTTP(w, req)
	return w
}

// Check status and parse response body
func CheckResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, output interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
}
