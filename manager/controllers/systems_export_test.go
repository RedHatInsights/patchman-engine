package controllers

import (
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeRequest(t *testing.T, path string, contentType string) *httptest.ResponseRecorder {
	core.SetupTest(t)
	return CreateRequest("GET", path, nil, contentType, SystemsExportHandler,
		core.ContextKV{Key: middlewares.KeyApiver, Value: 3})
}

func TestSystemsExportJSON(t *testing.T) {
	w := makeRequest(t, "/", "application/json")

	var output []SystemDBLookupV3
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 8, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, 2, output[0].SystemItemAttributesCommon.RhsaCount)
	assert.Equal(t, 2, output[0].SystemItemAttributesCommon.RhbaCount)
	assert.Equal(t, 1, output[0].SystemItemAttributesCommon.RheaCount)
	assert.Equal(t, "RHEL 8.10", output[0].SystemItemAttributesCommon.OS)
	assert.Equal(t, "8.10", output[0].SystemItemAttributesCommon.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributesCommon.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output[0].SystemItemAttributesCommon.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output[0].SystemItemAttributesCommon.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output[0].SystemItemAttributesCommon.Created.String())
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}, {"k2", "ns1", "val2"}}, output[0].SystemItemAttributesCommon.Tags) // nolint: lll
	assert.Equal(t, "baseline_1-1", output[0].SystemItemAttributesCommon.BaselineName)
	assert.Equal(t, int64(1), output[0].SystemItemAttributesV3Only.BaselineID)
}

func TestSystemsExportCSV(t *testing.T) {
	w := makeRequest(t, "/", "text/csv")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 10, len(lines))
	assert.Equal(t,
		"id,display_name,os,rhsm,tags,rhsa_count,rhba_count,rhea_count,other_count,packages_installed,baseline_name,"+
			"last_upload,stale_timestamp,stale_warning_timestamp,culled_timestamp,created,stale,baseline_id",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,RHEL 8.10,8.10,"+
		"\"[{'key':'k1','namespace':'ns1','value':'val1'},{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"2,2,1,0,0,baseline_1-1,2020-09-22T16:00:00Z,2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,"+
		"2018-08-26T16:00:00Z,false,1",
		lines[1])
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
		"id,display_name,os,rhsm,tags,rhsa_count,rhba_count,rhea_count,other_count,packages_installed,baseline_name,"+
			"last_upload,stale_timestamp,stale_warning_timestamp,culled_timestamp,created,stale,baseline_id",
		lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportSystemsTags(t *testing.T) {
	w := makeRequest(t, "/?tags=ns1/k2=val2", "application/json")

	var output []SystemDBLookupV3
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
		"/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC",
		"application/json",
	)

	var output []SystemDBLookupV3
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestSystemsExportBaselineFilter(t *testing.T) {
	w := makeRequest(t, "/?filter[baseline_name]=baseline_1-1", "application/json")

	var output []SystemDBLookupV3
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output[1].ID)
}
