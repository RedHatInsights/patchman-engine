package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeRequest(t *testing.T, path string, contentType string) *httptest.ResponseRecorder {
	core.SetupTest(t)
	return CreateRequest("GET", path, nil, &contentType, SystemsExportHandler)
}

func TestSystemsExportJSON(t *testing.T) {
	w := makeRequest(t, "/", "application/json")

	assert.Equal(t, http.StatusOK, w.Code)
	var output []SystemDBLookup

	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, 2, output[0].SystemItemAttributes.RhsaCount)
	assert.Equal(t, 3, output[0].SystemItemAttributes.RhbaCount)
	assert.Equal(t, 3, output[0].SystemItemAttributes.RheaCount)
	assert.Equal(t, "RHEL", output[0].SystemItemAttributes.OSName)
	assert.Equal(t, "8", output[0].SystemItemAttributes.OSMajor)
	assert.Equal(t, "10", output[0].SystemItemAttributes.OSMinor)
	assert.Equal(t, "RHEL 8.10", output[0].SystemItemAttributes.OS)
	assert.Equal(t, "8.10", output[0].SystemItemAttributes.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output[0].SystemItemAttributes.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.Created.String())
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}, {"k2", "ns1", "val2"}}, output[0].SystemItemAttributes.Tags)
	assert.Equal(t, "baseline_1-1", output[0].SystemItemAttributes.BaselineName)
	assert.Equal(t, true, *output[0].SystemItemAttributes.BaselineUpToDate)
}

func TestSystemsExportCSV(t *testing.T) {
	w := makeRequest(t, "/", "text/csv")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 10, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,"+
			"third_party,insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,os,"+
			"rhsm,stale_timestamp,stale_warning_timestamp,culled_timestamp,created,tags,baseline_name,baseline_uptodate",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,"+
		"2018-09-22T16:00:00Z,2020-09-22T16:00:00Z,2,3,3,0,false,true,00000000-0000-0000-0001-000000000001,0,0,RHEL,8,10,"+
		"RHEL 8.10,8.10,2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,2018-08-26T16:00:00Z,"+
		"\"[{'key':'k1','namespace':'ns1','value':'val1'},{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"baseline_1-1,true", lines[1])
}

func TestSystemsExportWrongFormat(t *testing.T) {
	w := makeRequest(t, "/", "test-format")

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	exp := `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`
	assert.Equal(t, exp, body)
}

func TestSystemsExportCSVFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[display_name]=nonexistant", "text/csv")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 2, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,"+
			"third_party,insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,os,rhsm,"+
			"stale_timestamp,stale_warning_timestamp,culled_timestamp,created,tags,baseline_name,baseline_uptodate",
		lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportSystemsTags(t *testing.T) {
	w := makeRequest(t, "/?tags=ns1/k2=val2", "application/json")

	assert.Equal(t, http.StatusOK, w.Code)
	var output []SystemDBLookup

	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestExportSystemsTagsInvalid(t *testing.T) {
	w := makeRequest(t, "/?tags=ns1/k3=val4&tags=invalidTag", "application/json")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsExportWorkloads(t *testing.T) {
	w := makeRequest(
		t,
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC",
		"application/json",
	)

	assert.Equal(t, http.StatusOK, w.Code)
	var output []SystemDBLookup

	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestSystemsExportBaselineFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[baseline_name]=baseline_1-1", "application/json")

	assert.Equal(t, http.StatusOK, w.Code)
	var output []SystemDBLookup
	ParseResponseBody(t, w.Body.Bytes(), &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output[1].ID)
}
