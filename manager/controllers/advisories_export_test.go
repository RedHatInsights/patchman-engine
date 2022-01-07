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

func TestAdvisoriesExportJSON(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "application/json")
	core.InitRouter(AdvisoriesExportHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var output []AdvisoryInlineItem
	ParseResponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, 9, len(output))
	assert.Equal(t, output[0].ID, "RH-1")
	assert.Equal(t, output[0].Description, "adv-1-des")
	assert.Equal(t, output[0].Synopsis, "adv-1-syn")
	assert.Equal(t, output[0].AdvisoryTypeName, "enhancement")
	assert.Equal(t, output[0].CveCount, 0)
	assert.Equal(t, output[0].RebootRequired, false)
	assert.Equal(t, output[0].ReleaseVersions, RelList{"7.0", "7Server"})
	assert.Equal(t, output[0].ApplicableSystems, 6)
}

func TestAdvisoriesExportCSV(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouter(AdvisoriesExportHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 11, len(lines))
	assert.Equal(t, "RH-1,adv-1-des,2016-09-22T16:00:00Z,adv-1-syn,1,enhancement,,0,false,\"7.0,7Server\",6", lines[1])
}

func TestAdvisoriesExportWrongFormat(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Accept", "test-format")
	core.InitRouter(AdvisoriesExportHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	body := w.Body.String()
	exp := `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`
	assert.Equal(t, exp, body)
}

func TestAdvisoriesExportCSVFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?filter[id]=RH-1", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouter(AdvisoriesExportHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	assert.Equal(t, 3, len(lines))
	assert.Equal(t, "RH-1,adv-1-des,2016-09-22T16:00:00Z,adv-1-syn,1,enhancement,,0,false,\"7.0,7Server\",6", lines[1])
	assert.Equal(t, "", lines[2])
}

func TestAdvisoriesExportTagsInvalid(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?tags=ns1/k3=val4&tags=invalidTag", nil)
	core.InitRouterWithPath(AdvisoriesExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestAdvisoriesExportIncorrectFilter(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?filter[filteriamnotexitst]=abcd", nil)
	req.Header.Add("Accept", "text/csv")
	core.InitRouterWithPath(AdvisoriesExportHandler, "/").ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
