package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageSystemsExportHandlerJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/kernel/systems?sort=id", nil, "application/json",
		PackageSystemsExportHandler, 3, "GET", "/:package_name/systems")

	var output []PackageSystemItemV3
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 3, len(output))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output[0].ID)
	assert.Equal(t, "5.6.13-200.fc31.x86_64", output[0].InstalledEVRA)
	assert.Equal(t, "5.10.13-200.fc31.x86_64", output[0].AvailableEVRA)
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}}, output[0].Tags)
	assert.Equal(t, "", output[0].BaselineName)
	assert.Equal(t, utils.PtrBoolNil(), output[0].BaselineUpToDate)
}

func TestPackageSystemsExportHandlerCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/kernel/systems?sort=id", nil, "text/csv",
		PackageSystemsExportHandler, 3, "GET", "/:package_name/systems")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 5, len(lines))
	assert.Equal(t, "id,display_name,installed_evra,available_evra,updatable,tags,"+
		"baseline_name,baseline_uptodate,baseline_id,os,rhsm,update_status,groups", lines[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000012,00000000-0000-0000-0000-000000000012,"+
		"5.6.13-200.fc31.x86_64,5.10.13-200.fc31.x86_64,true,"+
		"\"[{'key':'k1','namespace':'ns1','value':'val1'}]\",,,0,RHEL 8.1,8.1,Installable,[]",
		lines[1])
	assert.Equal(t, "00000000-0000-0000-0000-000000000013,00000000-0000-0000-0000-000000000013,"+
		"5.6.13-200.fc31.x86_64,,false,\"[{'key':'k1','namespace':'ns1','value':'val1'}]\",,,"+
		"0,RHEL 8.2,8.2,None,[]", lines[2])
}

func TestPackageSystemsExportInvalidName(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithParams("GET", "/unknown_package/systems", nil, "text/csv",
		PackageSystemsExportHandler, 3, "GET", "/:package_name/systems")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
