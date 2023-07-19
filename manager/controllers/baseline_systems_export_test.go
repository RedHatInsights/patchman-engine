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

func TestBaselineSystemsExportJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/1/systems", nil, "application/json", BaselineSystemsExportHandler,
		"/:baseline_id/systems")

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output))

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
	w := CreateRequestRouterWithPath("GET", "/1/systems", nil, "text/csv", BaselineSystemsExportHandler,
		"/:baseline_id/systems")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 4, len(lines))
	assert.Equal(t,
		"id,display_name,os,rhsm,installable_rhsa_count,installable_rhba_count,installable_rhea_count,"+
			"installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,"+
			"applicable_other_count,tags,groups,last_upload",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,RHEL 8.10,"+
		"8.10,2,2,1,0,2,3,3,0,\"[{'key':'k1','namespace':'ns1','value':'val1'},"+
		"{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"\"[{'id':'inventory-group-1','name':'group1'}]\",2020-09-22T16:00:00Z",
		lines[1])
}

func TestBaselineSystemsExportWrongFormat(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/1/systems", nil, "test-format", BaselineSystemsExportHandler,
		"/:baseline_id/systems")

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	assert.Equal(t, InvalidContentTypeErr, body)
}

func TestBaselineSystemsExportCSVFilter(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/1/systems?filter[display_name]=nonexistant", nil, "text/csv",
		BaselineSystemsExportHandler, "/:baseline_id/systems")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t,
		"id,display_name,os,rhsm,installable_rhsa_count,installable_rhba_count,installable_rhea_count,"+
			"installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,"+
			"applicable_other_count,tags,groups,last_upload",
		lines[0])
	assert.Equal(t, "", lines[1])
}

func TestExportBaselineSystemsTags(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/1/systems?tags=ns1/k2=val2", nil, "application/json",
		BaselineSystemsExportHandler, "/:baseline_id/systems")

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}

func TestExportBaselineSystemsTagsInvalid(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/1/systems?tags=ns1/k3=val4&tags=invalidTag", nil, "application/json",
		BaselineSystemsExportHandler, "/:baseline_id/systems")

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestBaselineSystemsExportWorkloads(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath(
		"GET",
		"/1/systems?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC",
		nil,
		"application/json",
		BaselineSystemsExportHandler,
		"/:baseline_id/systems",
	)

	var output []BaselineSystemsDBLookup
	CheckResponse(t, w, http.StatusOK, &output)

	assert.Equal(t, 2, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output[0].ID)
}
