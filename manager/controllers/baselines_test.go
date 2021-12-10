package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBaselines(t *testing.T, url string) BaselinesResponse {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	core.InitRouter(BaselinesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var output BaselinesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	return output
}

//nolint:dupl
func TestBaselinesDefault(t *testing.T) {
	output := testBaselines(t, "/")

	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 2, output.Data[0].ID)
	assert.Equal(t, "baseline", output.Data[0].Type)
	assert.Equal(t, "baseline_1-2", output.Data[0].Attributes.Name)
	assert.Equal(t, 2, output.Data[0].Attributes.Systems)

	// links
	assert.Equal(t, "/api/patch/v1/baselines?offset=0&limit=20&sort=-name", output.Links.First)
	assert.Equal(t, "/api/patch/v1/baselines?offset=0&limit=20&sort=-name", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestBaselinesOffsetLimit(t *testing.T) {
	output := testBaselines(t, "/?offset=0&limit=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 1, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestBaselinesUnlimited(t *testing.T) {
	output := testBaselines(t, "/?offset=0&limit=-1")
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, -1, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestBaselinesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=10&limit=4", nil)
	core.InitRouter(BaselinesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestBaselinesFilterTypeID(t *testing.T) {
	output := testBaselines(t, "/?filter[id]=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 1, output.Data[0].ID)
	assert.Equal(t, "baseline_1-1", output.Data[0].Attributes.Name)

	assert.Equal(t,
		"/api/patch/v1/baselines?offset=0&limit=20&filter[id]=eq:1&sort=-name",
		output.Links.First)
}

func TestBaselinesFilterName(t *testing.T) {
	output := testBaselines(t, "/?filter[name]=baseline_1-1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 1, output.Data[0].ID)
	assert.Equal(t, "baseline_1-1", output.Data[0].Attributes.Name)
}

func TestBaselinesFilterSystems(t *testing.T) {
	output := testBaselines(t, "/?filter[systems]=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 1, output.Data[0].ID)
	assert.Equal(t, "baseline_1-1", output.Data[0].Attributes.Name)
}

func TestBaselinesSort(t *testing.T) {
	output := testBaselines(t, "/?sort=name")

	assert.Equal(t, 1, len(output.Meta.Sort))
	assert.Equal(t, output.Meta.Sort[0], "name")
}

func TestBaselinesWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

//nolint:dupl
func TestBaselinesSearch(t *testing.T) {
	output := testBaselines(t, "/?search=baseline_1-1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 1, output.Data[0].ID)
	assert.Equal(t, "baseline", output.Data[0].Type)
	assert.Equal(t, "baseline_1-1", output.Data[0].Attributes.Name)
	assert.Equal(t, 1, output.Data[0].Attributes.Systems)

	// links
	assert.Equal(t, "/api/patch/v1/baselines?offset=0&limit=20&sort=-name&search=baseline_1-1",
		output.Links.First)
	assert.Equal(t, "/api/patch/v1/baselines?offset=0&limit=20&sort=-name&search=baseline_1-1",
		output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestBaselinesSearchFilter(t *testing.T) {
	output := testBaselines(t, "/?search=baseline_1-1&filter[systems]=2")
	assert.Equal(t, 0, len(output.Data))
}
