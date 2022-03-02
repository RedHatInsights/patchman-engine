package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/manager.env")
}

func testAdvisories(t *testing.T, url string) AdvisoriesResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var output AdvisoriesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)
	return output
}

//nolint:dupl
func TestAdvisoriesDefault(t *testing.T) {
	output := testAdvisories(t, "/")
	assert.Equal(t, 12, len(output.Data))
	assert.Equal(t, "RH-7", output.Data[0].ID, output.Data[0])
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)
	assert.Equal(t, false, output.Data[0].Attributes.RebootRequired)

	// links
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.First)
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 12, output.Meta.TotalItems)
}

func TestAdvisoriesOffsetLimit(t *testing.T) {
	output := testAdvisories(t, "/?offset=0&limit=2")
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 2, output.Meta.Limit)
	assert.Equal(t, 12, output.Meta.TotalItems)
}

func TestAdvisoriesUnlimited(t *testing.T) {
	output := testAdvisories(t, "/?offset=0&limit=-1")
	assert.Equal(t, 12, len(output.Data))
	assert.Equal(t, -1, output.Meta.Limit)
	assert.Equal(t, 12, output.Meta.TotalItems)
}

func TestAdvisoriesOffset(t *testing.T) {
	output := testAdvisories(t, "/?offset=1&limit=4")
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 1, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 12, output.Meta.TotalItems)
}

func TestAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestAdvisoriesOrderDate(t *testing.T) {
	output := testAdvisories(t, "/?sort=-public_date")
	// Advisory RH-7 has latest public date
	assert.Equal(t, 12, len(output.Data))
	assert.Equal(t, "RH-7", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)
}

func TestAdvisoriesOrderTypeID(t *testing.T) {
	output := testAdvisories(t, "/?sort=-advisory_type_name")
	assert.Equal(t, 12, len(output.Data))
	assert.Equal(t, "security", output.Data[0].Attributes.AdvisoryTypeName)
	assert.Equal(t, "security", output.Data[1].Attributes.AdvisoryTypeName)
	assert.Equal(t, "bugfix", output.Data[2].Attributes.AdvisoryTypeName)
	assert.Equal(t, "bugfix", output.Data[3].Attributes.AdvisoryTypeName)
	assert.Equal(t, "bugfix", output.Data[4].Attributes.AdvisoryTypeName)
	assert.Equal(t, "enhancement", output.Data[5].Attributes.AdvisoryTypeName)
	assert.Equal(t, "enhancement", output.Data[7].Attributes.AdvisoryTypeName)
	assert.Equal(t, "unspecified", output.Data[8].Attributes.AdvisoryTypeName)
	assert.Equal(t, "unspecified", output.Data[9].Attributes.AdvisoryTypeName)
	assert.Equal(t, "unknown", output.Data[10].Attributes.AdvisoryTypeName)
	assert.Equal(t, "unknown", output.Data[11].Attributes.AdvisoryTypeName)
}

// Ensure patched systems (ids: {7,8}) are not counted
func TestAdvisoriesPatchedMissing(t *testing.T) {
	output := testAdvisories(t, "/?sort=id")
	assert.Equal(t, 12, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[2].ID)
	assert.Equal(t, 6, output.Data[2].Attributes.ApplicableSystems)
}

func TestAdvisoriesFilterTypeID1(t *testing.T) {
	output := testAdvisories(t, "/?sort=id&filter[advisory_type_name]=enhancement")
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
	assert.Equal(t, "RH-4", output.Data[1].ID)
	assert.Equal(t, "RH-7", output.Data[2].ID)
	assert.Equal(t, FilterData{Values: []string{"enhancement"}, Operator: "eq"}, output.Meta.Filter["advisory_type_name"])
	assert.Equal(t,
		"/api/patch/v1/advisories?offset=0&limit=20&filter[advisory_type_name]=eq:enhancement&sort=id",
		output.Links.First)
}

func TestAdvisoriesFilterTypeID2(t *testing.T) {
	output := testAdvisories(t, "/?filter[advisory_type_name]=in:bugfix,enhancement")
	assert.Equal(t, 6, len(output.Data))
	assert.Equal(t, "RH-7", output.Data[0].ID)
	for _, a := range output.Data {
		assert.Contains(t, []string{"bugfix", "enhancement"}, a.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterTypeID3(t *testing.T) {
	output := testAdvisories(t, "/?filter[advisory_type_name]=in:unspecified,unknown")
	assert.Equal(t, 4, len(output.Data))
	for _, advisory := range output.Data {
		assert.Contains(t, "unknown unspecified", advisory.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterTypeID4(t *testing.T) {
	output := testAdvisories(t, "/?filter[advisory_type_name]=other")
	assert.Equal(t, 4, len(output.Data))
	for _, advisory := range output.Data {
		assert.Contains(t, "unknown unspecified", advisory.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterTypeID5(t *testing.T) {
	output := testAdvisories(t, "/?filter[advisory_type_name]!=other")
	assert.Equal(t, 4, len(output.Data))
	for _, advisory := range output.Data {
		assert.NotContains(t, "bugfix enhancement security", advisory.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterTypeID6(t *testing.T) {
	output := testAdvisories(t, "/?filter[advisory_type_name]=in:other,bugfix")
	assert.Equal(t, 7, len(output.Data))
	for _, advisory := range output.Data {
		assert.Contains(t, "bugfix unknown unspecified", advisory.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterTypeID7(t *testing.T) {
	output := testAdvisories(t, "/?sort=id&filter[advisory_type_name]=notin:other,bugfix")
	assert.Equal(t, 5, len(output.Data))
	for _, advisory := range output.Data {
		assert.Contains(t, "enhancement security", advisory.Attributes.AdvisoryTypeName)
	}
}

func TestAdvisoriesFilterApplicableSystems(t *testing.T) {
	output := testAdvisories(t, "/?filter[applicable_systems]=gt:1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "RH-1", output.Data[0].ID)
}

func TestAdvisoriesPossibleSorts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	for sort := range AdvisoriesFields {
		if sort == "ReleaseVersions" {
			// this fiesd is not sortable, skip it
			continue
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/?sort=%v", sort), nil)
		core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

		var output AdvisoriesResponse
		ParseResponseBody(t, w.Body.Bytes(), &output)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, len(output.Meta.Sort))
		assert.Equal(t, output.Meta.Sort[0], sort)
	}
}

func TestAdvisoriesWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

//nolint:dupl
func TestAdvisoriesSearch(t *testing.T) {
	output := testAdvisories(t, "/?search=h-3")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "RH-3", output.Data[0].ID)
	assert.Equal(t, "advisory", output.Data[0].Type)
	assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
	assert.Equal(t, "adv-3-des", output.Data[0].Attributes.Description)
	assert.Equal(t, "adv-3-syn", output.Data[0].Attributes.Synopsis)
	assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)

	// links
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date&search=h-3",
		output.Links.First)
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date&search=h-3",
		output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 1, output.Meta.TotalItems)
}

func TestAdvisoriesSearchCVE(t *testing.T) {
	output := testAdvisories(t, "/?search=CVE-2")
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "RH-6", output.Data[0].ID)
	assert.Equal(t, "RH-3", output.Data[1].ID)
}

func TestAdvisoriesSearchFilter(t *testing.T) {
	output := testAdvisories(t, "/?search=adv-3&filter[advisory_type_name]=other")
	assert.Equal(t, 0, len(output.Data))
}

func TestAdvisoriesTags(t *testing.T) {
	output := testAdvisories(t, "/?sort=id&tags=ns1/k2=val2")
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.ApplicableSystems)
	assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=id&tags=ns1/k2=val2", output.Links.First)
}

func TestListAdvisoriesTagsInvalid(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=invalidTag", nil)
	core.InitRouterWithPath(AdvisoriesListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func doTestWrongOffset(t *testing.T, path, q string, handler gin.HandlerFunc) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", q, nil)
	core.InitRouterWithParams(handler, 3, "GET", path).
		ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestAdvisoriesWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=1000", AdvisoriesListHandler)
}

func TestAdvisoryTagsInMetadata(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1?tags=ns1/k3=val4&tags=ns1/k1=val1", nil)
	core.InitRouterWithPath(AdvisoriesListHandler, "/:advisory_id").ServeHTTP(w, req)

	var output AdvisoriesResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)

	testMap := map[string]FilterData{
		"ns1/k1": {"eq", []string{"val1"}},
		"ns1/k3": {"eq", []string{"val4"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAdvisoryMetadataSums(t *testing.T) {
	output := testAdvisories(t, "/")
	var other, enhancement, bugfix, security int
	for _, ai := range output.Data {
		switch ai.Attributes.AdvisoryType {
		case 1:
			enhancement++
		case 2:
			bugfix++
		case 3:
			security++
		default:
			other++
		}
	}
	st := output.Meta.SubTotals
	assert.Equal(t, st["other"], other)
	assert.Equal(t, st["enhancement"], enhancement)
	assert.Equal(t, st["bugfix"], bugfix)
	assert.Equal(t, st["security"], security)
}
