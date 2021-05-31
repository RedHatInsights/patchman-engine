package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, 2, output[0].SystemItemAttributes.RhsaCount)
	assert.Equal(t, 3, output[0].SystemItemAttributes.RhbaCount)
	assert.Equal(t, 3, output[0].SystemItemAttributes.RheaCount)
	assert.Equal(t, "RHEL", output[0].SystemItemAttributes.OSName)
	assert.Equal(t, "8", output[0].SystemItemAttributes.OSMajor)
	assert.Equal(t, "1", output[0].SystemItemAttributes.OSMinor)
	assert.Equal(t, "8.1", output[0].SystemItemAttributes.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output[0].SystemItemAttributes.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.Created.String())
}

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
			"third_party,insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,rhsm,"+
			"stale_timestamp,stale_warning_timestamp,culled_timestamp,created",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,"+
		"2018-09-22T16:00:00Z,2020-09-22T16:00:00Z,2,3,3,false,true,00000000-0000-0000-0001-000000000001,0,0,RHEL,8,1,8.1,"+
		"2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,2018-08-26T16:00:00Z", lines[1])
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
			"third_party,insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,rhsm,"+
			"stale_timestamp,stale_warning_timestamp,culled_timestamp,created",
		lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportSystemsTags(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k2=val2", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithPath(SystemsExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output []SystemInlineItem

	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestExportSystemsTagsInvalid(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=invalidTag", nil)
	core.InitRouterWithPath(SystemsExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsExportWorkloads(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET",
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithPath(SystemsExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output []SystemInlineItem

	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}
