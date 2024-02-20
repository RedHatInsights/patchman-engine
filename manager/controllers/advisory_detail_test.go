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
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:advisory_id", "RH-9", "", nil, "", AdvisoryDetailHandler, core.V2APICtx)

	var outputV2 AdvisoryDetailResponse
	CheckResponse(t, w, http.StatusOK, &outputV2)
	// data
	outputV2.checkRH9Fields(t)
}

func (r *AdvisoryDetailResponse) checkRH9Fields(t *testing.T) {
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
		t, packages{"firefox-77.0.1-1.fc31.x86_64", "firefox-77.0.1-1.fc31.s390"},
		r.Data.Attributes.Packages,
	)
	assert.Equal(t, false, r.Data.Attributes.RebootRequired)
	assert.Equal(t, []string{"8.2", "8.4"}, r.Data.Attributes.ReleaseVersions)
	assert.Nil(t, r.Data.Attributes.Severity)
}

func TestAdvisoryDetailCVE(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:advisory_id", "RH-3", "", nil, "", AdvisoryDetailHandler, core.V2APICtx)

	var outputV2 AdvisoryDetailResponse
	CheckResponse(t, w, http.StatusOK, &outputV2)
	assert.Equal(t, 2, len(outputV2.Data.Attributes.Cves))
	assert.Equal(t, "CVE-1", outputV2.Data.Attributes.Cves[0])
	assert.Equal(t, "CVE-2", outputV2.Data.Attributes.Cves[1])
}

func TestAdvisoryNoIdProvided(t *testing.T) {
	core.SetupTest(t)
	var errResp utils.ErrorResponse
	w := CreateRequest("GET", "/", nil, "", AdvisoryDetailHandler,
		core.V1APICtx)

	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)

	w = CreateRequest("GET", "/", nil, "", AdvisoryDetailHandler,
		core.V2APICtx)
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, "advisory_id param not found", errResp.Error)
}

func TestAdvisoryNotFound(t *testing.T) {
	core.SetupTest(t)

	var errResp utils.ErrorResponse
	w := CreateRequestRouterWithPath("GET", "/:advisory_id", "foo", "", nil, "", AdvisoryDetailHandler, core.V1APICtx)

	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)

	w = CreateRequestRouterWithPath("GET", "/:advisory_id", "foo", "", nil, "", AdvisoryDetailHandler, core.V2APICtx)

	CheckResponse(t, w, http.StatusNotFound, &errResp)
	assert.Equal(t, "advisory not found", errResp.Error)
}

func testReqV2() *httptest.ResponseRecorder {
	return CreateRequestRouterWithPath("GET", "/:advisory_id", "RH-9", "", nil, "", AdvisoryDetailHandler,
		core.V2APICtx)
}

func TestAdvisoryDetailCached(t *testing.T) {
	core.SetupTest(t)

	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	testReqV2()      // load from db and save to cache
	w := testReqV2() // load from cache

	var outputV2 AdvisoryDetailResponse
	CheckResponse(t, w, http.StatusOK, &outputV2)
	outputV2.checkRH9Fields(t)
	assert.Equal(t, "found in cache", hook.LogEntries[4].Message)
}

func TestAdvisoryDetailCachePreloading(t *testing.T) {
	core.SetupTest(t)

	advisoryDetailCache.Purge()
	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	PreloadAdvisoryCacheItems()

	_, ok := advisoryDetailCache.Get("RH-8") // ensure some advisory in cache
	assert.True(t, ok)
	advisoryDetailCache.Purge()
}

func TestAdvisoryDetailFiltering(t *testing.T) {
	core.SetupTest(t)

	var errResp utils.ErrorResponse
	w := CreateRequestRouterWithPath("GET", "/:advisory_id", "RH-9", "?filter[filter]=abcd", nil, "",
		AdvisoryDetailHandler, core.V1APICtx)

	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)

	w = CreateRequestRouterWithPath("GET", "/:advisory_id", "RH-9", "?filter[filter]=abcd", nil, "",
		AdvisoryDetailHandler, core.V2APICtx)

	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, FilterNotSupportedMsg, errResp.Error)
}
