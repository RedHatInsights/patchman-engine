package controllers // nolint: dupl

import (
	"app/base/core"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemAdvisoriesExportJSON(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001", nil,
		"application/json", SystemAdvisoriesExportHandler, "/:inventory_id")

	var output []AdvisoryInlineItemV3
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 8, len(output))
	assert.Equal(t, output[0].Description, "adv-1-des")
}

func TestSystemAdvisoriesExportCSV(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/00000000-0000-0000-0000-000000000001", nil,
		"text/csv", SystemAdvisoriesExportHandler, "/:inventory_id")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 10, len(lines))
	assert.Equal(t, "id,description,public_date,synopsis,advisory_type,advisory_type_name,severity,cve_count,"+
		"reboot_required,release_versions", lines[0])
	assert.Equal(t, "RH-1,adv-1-des,2016-09-22T16:00:00Z,adv-1-syn,1,enhancement,,0,false,\"7.0,7Server\"", lines[1])
}

func TestUnknownSystemAdvisoriesExport(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/unknownsystem", nil, "text/csv", SystemAdvisoriesExportHandler,
		"/:inventory_id")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
