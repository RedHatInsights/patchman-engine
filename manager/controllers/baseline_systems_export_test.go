package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint: dupl
func TestBaselineSystemsExportJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "", nil, "application/json",
		BaselineSystemsExportHandler)

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 3, len(output))

	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].DisplayName)
	assert.Equal(t, "RHEL 8.10", output[0].OS)
	assert.Equal(t, "8.10", output[0].Rhsm)
	assert.Equal(t, 2, output[0].InstallableRhsaCount)
	assert.Equal(t, 2, output[0].InstallableRhbaCount)
	assert.Equal(t, 1, output[0].InstallableRheaCount)
	assert.Equal(t, 0, output[0].InstallableOtherCount)
	assert.Equal(t, 2, output[0].ApplicableRhsaCount)
	assert.Equal(t, 3, output[0].ApplicableRhbaCount)
	assert.Equal(t, 3, output[0].ApplicableRheaCount)
	assert.Equal(t, 0, output[0].ApplicableOtherCount)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output[1].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output[1].DisplayName)
}

func TestBaselineSystemsExportCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "", nil, "text/csv",
		BaselineSystemsExportHandler)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\r\n")

	assert.Equal(t, 5, len(lines))
	assert.Equal(t, TemplateCsvHeader, lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,RHEL 8.10,"+
		"8.10,2,2,1,0,2,3,3,0,\"[{'key':'k1','namespace':'ns1','value':'val1'},"+
		"{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"\"[{'id':'inventory-group-1','name':'group1'}]\",2020-09-22T16:00:00Z",
		lines[1])
}

func TestBaselineSystemsExportWrongFormat(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "", nil, "test-format",
		BaselineSystemsExportHandler)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	assert.Equal(t, InvalidContentTypeErr, body)
}

func TestBaselineSystemsExportCSVFilter(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "?filter[display_name]=nonexistant", nil,
		"text/csv", BaselineSystemsExportHandler)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\r\n")

	assert.Equal(t, TemplateCsvHeader, lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportBaselineSystemsTags(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "?tags=ns1/k2=val2", nil, "application/json",
		BaselineSystemsExportHandler)

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestExportBaselineSystemsTagsInvalid(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "?tags=ns1/k3=val4&tags=invalidTag", nil,
		"application/json", BaselineSystemsExportHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestBaselineSystemsExportWorkloads(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1",
		"?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids]=ABC", nil, "application/json",
		BaselineSystemsExportHandler)

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}
