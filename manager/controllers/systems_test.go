package controllers

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// nolint: lll
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
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2020-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, "2018-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastEvaluation.String())
	assert.Equal(t, 3, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 3, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, 0, output.Data[0].Attributes.PackagesInstalled)
	assert.Equal(t, 0, output.Data[0].Attributes.PackagesUpdatable)

	// links
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.First)
	assert.Equal(t, "/api/patch/v1/systems?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// test meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
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

func TestSystemsOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	core.InitRouter(SystemsListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
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
	req, _ := http.NewRequest("GET", "/?tags=ns1/k1=val1", nil)
	core.InitRouterWithPath(SystemsListHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads(t *testing.T) {
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

func TestSystemsExportJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithPath(SystemsExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output []SystemInlineItem

	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output))
	assert.Equal(t, output[0].ID, "00000000-0000-0000-0000-000000000001")
}

// nolint: lll
func TestSystemsExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouter(SystemsExportHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 10, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,stale,"+
			"packages_installed,packages_updatable",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,2018-09-22T16:00:00Z,2020-09-22T16:00:00Z,2,3,3,false,0,0", lines[1])
}

func TestSystemsExportWrongFormat(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "test-format")
	core.InitRouter(SystemsExportHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	exp := `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`
	assert.Equal(t, exp, body)
}

func TestSystemsExportCSVFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?filter[display_name]=nonexistant", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouter(SystemsExportHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 2, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,stale,"+
			"packages_installed,packages_updatable",
		lines[0])
	assert.Equal(t, "", lines[1])
}
