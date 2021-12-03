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
	assert.Equal(t, http.StatusOK, w.Code)
	var output []SystemDBLookup
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 6, len(output))
	assert.Equal(t, output[0].ID, "00000000-0000-0000-0000-000000000001")
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}, {"k2", "ns1", "val2"}}, output[0].SystemItemAttributes.Tags)
}

func TestAdvisorySystemsExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/RH-1", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithPath(AdvisorySystemsExportHandler, "/:advisory_id").ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 8, len(lines))
	assert.Equal(t,
		"id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,stale,third_party,"+
			"insights_id,packages_installed,packages_updatable,os_name,os_major,os_minor,os,rhsm,stale_timestamp,"+
			"stale_warning_timestamp,culled_timestamp,created,tags,baseline_name,baseline_uptodate", lines[0])

	assert.Equal(t, "00000000-0000-0000-0000-000000000001,00000000-0000-0000-0000-000000000001,"+
		"2018-09-22T16:00:00Z,2020-09-22T16:00:00Z,2,3,3,0,false,true,00000000-0000-0000-0001-000000000001,0,0,"+
		"RHEL,8,10,RHEL 8.10,8.10,2018-08-26T16:00:00Z,2018-09-02T16:00:00Z,2018-09-09T16:00:00Z,2018-08-26T16:00:00Z,"+
		"\"[{'key':'k1','namespace':'ns1','value':'val1'},{'key':'k2','namespace':'ns1','value':'val2'}]\","+
		"baseline_1-1,true", lines[1])
}
