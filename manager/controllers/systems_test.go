package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemsDefault(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	// data
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0001-000000000001", output.Data[0].Attributes.InsightsID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2020-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastEvaluation.String())
	assert.Equal(t, 3, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 3, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, 0, output.Data[0].Attributes.PackagesInstalled)
	assert.Equal(t, 0, output.Data[0].Attributes.PackagesUpdatable)
	assert.Equal(t, "RHEL", output.Data[0].Attributes.OSName)
	assert.Equal(t, "8", output.Data[0].Attributes.OSMajor)
	assert.Equal(t, "1", output.Data[0].Attributes.OSMinor)
	assert.Equal(t, "8.1", output.Data[0].Attributes.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output.Data[0].Attributes.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output.Data[0].Attributes.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output.Data[0].Attributes.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output.Data[0].Attributes.Created.String())

	// links
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.First)
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// test meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
	assert.Equal(t, 8, output.Meta.SubTotals["patched"])
	assert.Equal(t, 0, output.Meta.SubTotals["unpatched"])
	assert.Equal(t, 0, output.Meta.SubTotals["stale"])
}

func TestSystemsOffsetLimit(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=0&limit=4", nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsOffset(t *testing.T) { //nolint:dupl
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=4&limit=4", nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 4, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=13&limit=4", SystemsListHandler)
}

func TestSystemsWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSystemsSearch(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?search=001", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)
}

func TestSystemsTags(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k2=val2", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsTagsMultiple(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=ns1/k1=val1", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsUnknown(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=ns1/k1=unk", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsNoVal(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=ns1/k1", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsInvalid(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=invalidTag", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsWorkloads1(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET",
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads2(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET",
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][]=ABC", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads3(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET",
		"/?filter[system_profile][sap_system]=false", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsPackagesCount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=-packages_installed,id", nil)
	core.InitRouterWithAccount(SystemsListHandler, "/", 3).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].Attributes.DisplayName)
	assert.Equal(t, 2, output.Data[0].Attributes.PackagesInstalled)
	assert.Equal(t, 2, output.Data[0].Attributes.PackagesUpdatable)
}

func doTestSystemsFilter(t *testing.T, url string) SystemsResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	return output
}

func TestSystemsFilterAdvCount1(t *testing.T) {
	output := doTestSystemsFilter(t, "/?filter[rhba_count]=3")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 3, output.Data[0].Attributes.RhbaCount)
}

func TestSystemsFilterAdvCount2(t *testing.T) {
	output := doTestSystemsFilter(t, "/?filter[rhea_count]=3")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 3, output.Data[0].Attributes.RheaCount)
}

func TestSystemsFilterAdvCount3(t *testing.T) {
	output := doTestSystemsFilter(t, "/?filter[rhsa_count]=2")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
}

func TestSystemsFilterAdvCount4(t *testing.T) {
	output := doTestSystemsFilter(t, "/?filter[other_count]=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 1, output.Data[0].Attributes.OtherCount)
}
