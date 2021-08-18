package controllers // nolint: dupl

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdvisorySystemsExportJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouterWithPath(AdvisorySystemsExportHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var output []AdvisoryInlineItem
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 6, len(output))
	assert.Equal(t, output[0].ID, "00000000-0000-0000-0000-000000000001")
}

func TestAdvisorySystemsExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithPath(AdvisorySystemsExportHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 8, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,third_party,"+
			"insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,rhsm,stale_timestamp,"+
			"stale_warning_timestamp,culled_timestamp,created",
		lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,"+
		"2018-09-22T16:00:00Z,2020-09-22T16:00:00Z,2,3,3,0,false,true,00000000-0000-0000-0001-000000000001,0,0,"+
		"RHEL,8,1,8.1,2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,2018-08-26T16:00:00Z", lines[1])
}
