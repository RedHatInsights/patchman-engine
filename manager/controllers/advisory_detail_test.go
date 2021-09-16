package controllers

import (
	"app/base/core"
	"app/base/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdvisoryDetailDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-9", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	checkRH9Fields(t, output)
}

func checkRH9Fields(t *testing.T, response AdvisoryDetailResponse) {
	assert.Equal(t, "advisory", response.Data.Type)
	assert.Equal(t, "RH-9", response.Data.ID)
	assert.Equal(t, "adv-9-syn", response.Data.Attributes.Synopsis)
	assert.Equal(t, "adv-9-des", response.Data.Attributes.Description)
	assert.Equal(t, "adv-9-sol", response.Data.Attributes.Solution)
	assert.Equal(t, "2016-09-22 20:00:00 +0000 UTC", response.Data.Attributes.PublicDate.String())
	assert.Equal(t, "2018-09-22 20:00:00 +0000 UTC", response.Data.Attributes.ModifiedDate.String())
	assert.Equal(t, 1, len(response.Data.Attributes.Packages))
	assert.Equal(t, "77.0.1-1.fc31.x86_64", response.Data.Attributes.Packages["firefox"])
	assert.Equal(t, false, response.Data.Attributes.RebootRequired)
	assert.Nil(t, response.Data.Attributes.Severity)
}

func TestAdvisoryDetailCVE(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-3", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data.Attributes.Cves))
	assert.Equal(t, "CVE-1", output.Data.Attributes.Cves[0])
	assert.Equal(t, "CVE-2", output.Data.Attributes.Cves[1])
}

func TestAdvisoryNoIdProvided(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouter(AdvisoryDetailHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)
}

func TestAdvisoryNotFound(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)
}

func testReq() *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-9", nil)
	core.InitRouterWithPath(AdvisoryDetailHandler, "/:advisory_id").ServeHTTP(w, req)
	return w
}

func TestAdvisoryDetailCached(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	testReq()      // load from db and save to cache
	w := testReq() // load from cache

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	checkRH9Fields(t, output)
	assert.Equal(t, "found in cache", hook.LogEntries[4].Message)
}

func TestAdvisoryDetailCachePreloading(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	advisoryDetailCache.Purge()
	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	PreloadAdvisoryCacheItems()

	_, ok := advisoryDetailCache.Get("RH-8") // ensure some advisory in cache
	assert.True(t, ok)
	advisoryDetailCache.Purge()
}
