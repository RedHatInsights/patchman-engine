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

var SystemCsvHeader = "id,display_name,os,rhsm,tags,last_evaluation," +
	"rhsa_count,rhba_count,rhea_count,other_count,packages_installed," +
	"baseline_name,last_upload,stale_timestamp,stale_warning_timestamp,culled_timestamp,created,stale," +
	"satellite_managed,built_pkgcache,packages_installable,packages_applicable," +
	"installable_rhsa_count,installable_rhba_count,installable_rhea_count,installable_other_count," +
	"applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,applicable_other_count," +
	"baseline_id,template_name,template_uuid,groups,arch"

func makeRequest(t *testing.T, path string, contentType string) *httptest.ResponseRecorder {
	core.SetupTest(t)
	return CreateRequest("GET", path, nil, contentType, SystemsExportHandler)
}

func TestSystemsExportJSON(t *testing.T) {
	w := makeRequest(t, "/", "application/json")

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 9, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, 2, output[0].SystemItemAttributes.RhsaCount)
	assert.Equal(t, 2, output[0].SystemItemAttributes.RhbaCount)
	assert.Equal(t, 1, output[0].SystemItemAttributes.RheaCount)
	assert.Equal(t, "RHEL 8.10", output[0].SystemItemAttributes.OS)
	assert.Equal(t, "8.10", output[0].SystemItemAttributes.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output[0].SystemItemAttributes.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output[0].SystemItemAttributes.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributes.Created.String())
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}, {"k2", "ns1", "val2"}}, output[0].SystemItemAttributes.Tags)
	assert.Equal(t, "baseline_1-1", output[0].SystemItemAttributes.BaselineName)
	assert.Equal(t, int64(1), output[0].SystemItemAttributes.BaselineID)
}

func TestSystemsExportCSV(t *testing.T) {
	w := makeRequest(t, "/", "text/csv")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\r\n")

	assert.Equal(t, 11, len(lines))
	assert.Equal(t, SystemCsvHeader, lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,RHEL 8.10,8.10,"+
		"\"[{'key':'k1','namespace':'ns1','value':'val1'},{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"2018-09-22T16:00:00Z,2,2,1,0,0,baseline_1-1,"+
		"2020-09-22T16:00:00Z,2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,2018-08-26T16:00:00Z,"+
		"false,false,false,0,0,2,2,1,0,2,3,3,3,1,temp1-1,99900000-0000-0000-0000-000000000001,"+
		"\"[{'id':'inventory-group-1','name':'group1'}]\",x86_64",
		lines[1])
}

func TestSystemsExportWrongFormat(t *testing.T) {
	w := makeRequest(t, "/", "test-format")

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	assert.Equal(t, InvalidContentTypeErr, body)
}

func TestSystemsExportCSVFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[display_name]=nonexistant", "text/csv")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\r\n")

	assert.Equal(t, 2, len(lines))
	assert.Equal(t, SystemCsvHeader, lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportSystemsTags(t *testing.T) {
	w := makeRequest(t, "/?tags=ns1/k2=val2", "application/json")

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestExportSystemsTagsInvalid(t *testing.T) {
	w := makeRequest(t, "/?tags=ns1/k3=val4&tags=invalidTag", "application/json")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsExportWorkloads(t *testing.T) {
	w := makeRequest(
		t,
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids]=ABC",
		"application/json",
	)

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestSystemsExportBaselineFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[baseline_name]=baseline_1-1", "application/json")

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output[1].ID)
}

func TestSystemsExportFilterPartialOS(t *testing.T) {
	w := makeRequest(t, "/?filter[osname]=RHEL&filter[osmajor]=8&filter[osminor]=1", "application/json")

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 3, len(output))
	for _, o := range output {
		assert.Equal(t, "RHEL 8.1", o.OS)
	}
}

func TestSystemsExportArchFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[arch]=x86_64", "application/json")

	var output []SystemDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 8, len(output))
	for _, o := range output {
		assert.Equal(t, "x86_64", o.Arch)
	}
}
