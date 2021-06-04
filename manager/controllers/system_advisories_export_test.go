package controllers // nolint: dupl

import (
	"app/base/core"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSystemAdvisoriesExportJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithPath(SystemAdvisoriesExportHandler, "/:inventory_id").ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var output []AdvisoryInlineItem
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 8, len(output))
	assert.Equal(t, output[0].Description, "adv-1-des")
}

func TestSystemAdvisoriesExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/00000000-0000-0000-0000-000000000001", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithPath(SystemAdvisoriesExportHandler, "/:inventory_id").ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 10, len(lines))
	assert.Equal(t, "id,description,public_date,synopsis,advisory_type,severity,cve_count,applicable_systems",
		lines[0])
	assert.Equal(t, "RH-1,adv-1-des,2016-09-22T16:00:00Z,adv-1-syn,1,,0,0", lines[1])
}
