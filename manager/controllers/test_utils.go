package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const InvalidContentTypeErr = `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`
const LatestAPIVersion = 3

var V3APICtx = ContextKV{Key: utils.KeyApiver, Value: 3}

var (
	testInventoryID1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testInventoryID2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testInventoryID4 = uuid.MustParse("00000000-0000-0000-0000-000000000004")
)

type ContextKV struct {
	Key   string
	Value any
}

var WorkspacesTestCtx = ContextKV{
	Key: utils.KeyInventoryWorkspaces,
	Value: []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-999999999999",
	},
}

func InitRouter(handler gin.HandlerFunc, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithPath(handler, "/", contextKVs...)
}

func InitRouterWithParams(handler gin.HandlerFunc, account int, method, path string,
	contextKVs ...ContextKV) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares.RequestResponseLogger())
	router.Use(middlewares.MockAuthenticator(account))
	router.Use(func(c *gin.Context) {
		// set default api version for tests to latest
		c.Set(utils.KeyApiver, LatestAPIVersion)
		c.Set(utils.KeyInventoryWorkspaces, WorkspacesTestCtx.Value)
		for _, kv := range contextKVs {
			c.Set(kv.Key, kv.Value)
		}
	})
	router.Handle(method, path, handler)
	return router
}

func InitRouterWithPath(handler gin.HandlerFunc, path string, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithParams(handler, 1, "GET", path, contextKVs...)
}

func InitRouterWithAccount(handler gin.HandlerFunc, path string, account int, contextKVs ...ContextKV) *gin.Engine {
	return InitRouterWithParams(handler, account, "GET", path, contextKVs...)
}

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
	contextKVs ...ContextKV) (
	w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, url, body, contentType)
	InitRouter(handler, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with params
func CreateRequestRouterWithParams(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerAccount int, contextKVs ...ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	InitRouterWithParams(handler, routerAccount, method, routerPath, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with path
func CreateRequestRouterWithPath(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, contextKVs ...ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	InitRouterWithPath(handler, routerPath, contextKVs...).ServeHTTP(w, req)
	return w
}

// Create request and initialize router with account
func CreateRequestRouterWithAccount(method, routerPath, param, queryString string, body io.Reader, contentType string,
	handler gin.HandlerFunc, routerAccount int, contextKVs ...ContextKV) (w *httptest.ResponseRecorder) {
	w, req := prepareRequest(method, getURLFromRouterPath(routerPath, param, queryString), body, contentType)
	InitRouterWithAccount(handler, routerPath, routerAccount, contextKVs...).ServeHTTP(w, req)
	return w
}

// Check status and parse response body
func CheckResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, output interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &output)
}
