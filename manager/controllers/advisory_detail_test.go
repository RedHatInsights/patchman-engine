package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAdvisoryDetailDefault(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/RH-9", nil, nil, AdvisoryDetailHandlerV1, "/:advisory_id")

	assert.Equal(t, http.StatusOK, w.Code)
	var outputV1 AdvisoryDetailResponseV1
	ParseResponseBody(t, w.Body.Bytes(), &outputV1)
	// data
	outputV1.checkRH9Fields(t)

	w = CreateRequestRouterWithPath("GET", "/RH-9", nil, nil, AdvisoryDetailHandlerV2, "/:advisory_id")

	assert.Equal(t, http.StatusOK, w.Code)
	var outputV2 AdvisoryDetailResponseV2
	ParseResponseBody(t, w.Body.Bytes(), &outputV2)
	// data
	outputV2.checkRH9Fields(t)
}

func (r *AdvisoryDetailResponseV1) checkRH9Fields(t *testing.T) {
	assert.Equal(t, "advisory", r.Data.Type)
	assert.Equal(t, "RH-9", r.Data.ID)
	assert.Equal(t, "adv-9-syn", r.Data.Attributes.Synopsis)
	assert.Equal(t, "adv-9-des", r.Data.Attributes.Description)
	assert.Equal(t, "adv-9-sol", *r.Data.Attributes.Solution)
	assert.Equal(t, "security", r.Data.Attributes.AdvisoryTypeName)
	assert.Equal(t, "2016-09-22 20:00:00 +0000 UTC", r.Data.Attributes.PublicDate.String())
	assert.Equal(t, "2018-09-22 20:00:00 +0000 UTC", r.Data.Attributes.ModifiedDate.String())
	assert.Equal(t, 1, len(r.Data.Attributes.Packages))
	assert.Equal(t, "77.0.1-1.fc31.x86_64", r.Data.Attributes.Packages["firefox"])
	assert.Equal(t, false, r.Data.Attributes.RebootRequired)
	assert.Equal(t, []string{"8.2", "8.4"}, r.Data.Attributes.ReleaseVersions)
	assert.Nil(t, r.Data.Attributes.Severity)
}

func (r *AdvisoryDetailResponseV2) checkRH9Fields(t *testing.T) {
	assert.Equal(t, "advisory", r.Data.Type)
	assert.Equal(t, "RH-9", r.Data.ID)
	assert.Equal(t, "adv-9-syn", r.Data.Attributes.Synopsis)
	assert.Equal(t, "adv-9-des", r.Data.Attributes.Description)
	assert.Equal(t, "adv-9-sol", *r.Data.Attributes.Solution)
	assert.Equal(t, "security", r.Data.Attributes.AdvisoryTypeName)
	assert.Equal(t, "2016-09-22 20:00:00 +0000 UTC", r.Data.Attributes.PublicDate.String())
	assert.Equal(t, "2018-09-22 20:00:00 +0000 UTC", r.Data.Attributes.ModifiedDate.String())
	assert.Equal(t, 2, len(r.Data.Attributes.Packages))
	assert.Equal(
		t, packagesV2{"firefox-77.0.1-1.fc31.x86_64", "firefox-77.0.1-1.fc31.s390"},
		r.Data.Attributes.Packages,
	)
	assert.Equal(t, false, r.Data.Attributes.RebootRequired)
	assert.Equal(t, []string{"8.2", "8.4"}, r.Data.Attributes.ReleaseVersions)
	assert.Nil(t, r.Data.Attributes.Severity)
}

func TestAdvisoryDetailCVE(t *testing.T) {
	SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/RH-3", nil, nil, AdvisoryDetailHandlerV1, "/:advisory_id")

	assert.Equal(t, http.StatusOK, w.Code)
	var outputV1 AdvisoryDetailResponseV1
	ParseResponseBody(t, w.Body.Bytes(), &outputV1)
	assert.Equal(t, 2, len(outputV1.Data.Attributes.Cves))
	assert.Equal(t, "CVE-1", outputV1.Data.Attributes.Cves[0])
	assert.Equal(t, "CVE-2", outputV1.Data.Attributes.Cves[1])

	w = CreateRequestRouterWithPath("GET", "/RH-3", nil, nil, AdvisoryDetailHandlerV2, "/:advisory_id")

	assert.Equal(t, http.StatusOK, w.Code)
	var outputV2 AdvisoryDetailResponseV2
	ParseResponseBody(t, w.Body.Bytes(), &outputV2)
	assert.Equal(t, 2, len(outputV2.Data.Attributes.Cves))
	assert.Equal(t, "CVE-1", outputV2.Data.Attributes.Cves[0])
	assert.Equal(t, "CVE-2", outputV2.Data.Attributes.Cves[1])
}

func TestAdvisoryNoIdProvided(t *testing.T) {
	SetupTest(t)

	var errResp utils.ErrorResponse
	w := CreateRequest("GET", "/", nil, nil, AdvisoryDetailHandlerV1)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)

	w = CreateRequest("GET", "/", nil, nil, AdvisoryDetailHandlerV2)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)
}

func TestAdvisoryNotFound(t *testing.T) {
	SetupTest(t)

	var errResp utils.ErrorResponse
	w := CreateRequestRouterWithPath("GET", "/foo", nil, nil, AdvisoryDetailHandlerV1, "/:advisory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)

	w = CreateRequestRouterWithPath("GET", "/foo", nil, nil, AdvisoryDetailHandlerV2, "/:advisory_id")

	assert.Equal(t, http.StatusNotFound, w.Code)
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)
}

func testReqV1() *httptest.ResponseRecorder {
	return CreateRequestRouterWithPath("GET", "/RH-9", nil, nil, AdvisoryDetailHandlerV1, "/:advisory_id")
}

func testReqV2() *httptest.ResponseRecorder {
	return CreateRequestRouterWithPath("GET", "/RH-9", nil, nil, AdvisoryDetailHandlerV2, "/:advisory_id")
}

func TestAdvisoryDetailCached(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	testReqV1()      // load from db and save to cache
	w := testReqV1() // load from cache

	assert.Equal(t, http.StatusOK, w.Code)
	var output AdvisoryDetailResponseV1
	ParseResponseBody(t, w.Body.Bytes(), &output)
	output.checkRH9Fields(t)
	assert.Equal(t, "found in cache", hook.LogEntries[4].Message)

	testReqV2()     // load from db and save to cache
	w = testReqV2() // load from cache

	assert.Equal(t, http.StatusOK, w.Code)
	var outputV2 AdvisoryDetailResponseV2
	ParseResponseBody(t, w.Body.Bytes(), &outputV2)
	output.checkRH9Fields(t)
	assert.Equal(t, "found in cache", hook.LogEntries[7].Message)
}

func TestAdvisoryDetailCachePreloading(t *testing.T) {
	SetupTest(t)

	advisoryDetailCacheV1.Purge()
	advisoryDetailCacheV2.Purge()
	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	PreloadAdvisoryCacheItems()

	_, ok := advisoryDetailCacheV1.Get("RH-8") // ensure some advisory in cache
	assert.True(t, ok)
	advisoryDetailCacheV1.Purge()
	_, ok = advisoryDetailCacheV2.Get("RH-8") // ensure some advisory in cache
	assert.True(t, ok)
	advisoryDetailCacheV2.Purge()
}

func TestAdvisoryDetailFiltering(t *testing.T) {
	SetupTest(t)

	var errResp utils.ErrorResponse
	w := CreateRequestRouterWithPath("GET", "/RH-9?filter[filter]=abcd", nil, nil, AdvisoryDetailHandlerV1,
		"/:advisory_id")

	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)

	w = CreateRequestRouterWithPath("GET", "/RH-9?filter[filter]=abcd", nil, nil, AdvisoryDetailHandlerV2,
		"/:advisory_id")

	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)
}
