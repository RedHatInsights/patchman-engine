package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/manager.env")
}

// nolint: unparam
func testAdvisoriesOk(t *testing.T, method, url string, check func(out AdvisoriesResponse)) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output AdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	check(output)
}

//nolint:dupl
func TestAdvisoriesDefault(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/", func(output AdvisoriesResponse) {
		assert.Equal(t, 8, len(output.Data))
		assert.Equal(t, "RH-7", output.Data[0].ID, output.Data[0])
		assert.Equal(t, "advisory", output.Data[0].Type)
		assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
		assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
		assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
		assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)

		// links
		assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.First)
		assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.Last)
		assert.Nil(t, output.Links.Next)
		assert.Nil(t, output.Links.Previous)

		// meta
		assert.Equal(t, 0, output.Meta.Offset)
		assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
		assert.Equal(t, 8, output.Meta.TotalItems)
	})
}

func TestAdvisoriesOffsetLimit(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?offset=0&limit=2", func(output AdvisoriesResponse) {
		assert.Equal(t, 2, len(output.Data))
		assert.Equal(t, 0, output.Meta.Offset)
		assert.Equal(t, 2, output.Meta.Limit)
		assert.Equal(t, 8, output.Meta.TotalItems)
	})
}

func TestAdvisoriesUnlimited(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?offset=0&limit=-1", func(output AdvisoriesResponse) {
		assert.Equal(t, 8, len(output.Data))
		assert.Equal(t, -1, output.Meta.Limit)
		assert.Equal(t, 8, output.Meta.TotalItems)
	})
}

func TestAdvisoriesOffset(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?offset=1&limit=4", func(output AdvisoriesResponse) {
		assert.Equal(t, 4, len(output.Data))
		assert.Equal(t, 1, output.Meta.Offset)
		assert.Equal(t, 4, output.Meta.Limit)
		assert.Equal(t, 8, output.Meta.TotalItems)
	})
}

func TestAdvisoriesOffsetOverflow(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?offset=13&limit=4", nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp utils.ErrorResponse
	ParseReponseBody(t, w.Body.Bytes(), &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestAdvisoriesOrder(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?sort=-public_date", func(output AdvisoriesResponse) {
		// Advisory RH-7 has latest public date
		assert.Equal(t, 8, len(output.Data))
		assert.Equal(t, "RH-7", output.Data[0].ID)
		assert.Equal(t, "advisory", output.Data[0].Type)
		assert.Equal(t, "2017-09-22 19:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
		assert.Equal(t, "adv-7-des", output.Data[0].Attributes.Description)
		assert.Equal(t, "adv-7-syn", output.Data[0].Attributes.Synopsis)
		assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)
	})
}

func TestAdvisoriesFilter(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?filter[advisory_type]=1", func(output AdvisoriesResponse) {
		assert.Equal(t, 3, len(output.Data))
		assert.Equal(t, "RH-7", output.Data[0].ID)
		assert.Equal(t, "RH-1", output.Data[1].ID)
		assert.Equal(t, "RH-4", output.Data[2].ID)

		assert.Equal(t, FilterData{Values: []string{"1"}, Operator: "eq"}, output.Meta.Filter["advisory_type"])

		assert.Equal(t,
			"/api/patch/v1/advisories?offset=0&limit=20&filter[advisory_type]=eq:1&sort=-public_date",
			output.Links.First)
	})

	testAdvisoriesOk(t, "GET", "/?filter[applicable_systems]=gt:1", func(output AdvisoriesResponse) {
		assert.Equal(t, 1, len(output.Data))
		assert.Equal(t, "RH-1", output.Data[0].ID)
	})

	testAdvisoriesOk(t, "GET", "/?filter[advisory_type]=in:1,2", func(output AdvisoriesResponse) {
		assert.Equal(t, 6, len(output.Data))
		assert.Equal(t, "RH-7", output.Data[0].ID)

		for _, a := range output.Data {
			assert.Contains(t, []int{1, 2}, a.Attributes.AdvisoryType)
		}
	})
}

func TestAdvisoriesPossibleSorts(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	for sort := range AdvisoriesFields {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/?sort=%v", sort), nil)
		core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

		var output AdvisoriesResponse
		ParseReponseBody(t, w.Body.Bytes(), &output)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, 1, len(output.Meta.Sort))
		assert.Equal(t, output.Meta.Sort[0], sort)
	}
}

func TestAdvisoriesWrongSort(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?sort=unknown_key", nil)
	core.InitRouter(AdvisoriesListHandler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

//nolint:dupl
func TestAdvisoriesSearch(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?search=h-3", func(output AdvisoriesResponse) {
		assert.Equal(t, 1, len(output.Data))
		assert.Equal(t, "RH-3", output.Data[0].ID)
		assert.Equal(t, "advisory", output.Data[0].Type)
		assert.Equal(t, "2016-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.PublicDate.String())
		assert.Equal(t, "adv-3-des", output.Data[0].Attributes.Description)
		assert.Equal(t, "adv-3-syn", output.Data[0].Attributes.Synopsis)
		assert.Equal(t, 1, output.Data[0].Attributes.ApplicableSystems)

		// links
		assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.First)
		assert.Equal(t, "/api/patch/v1/advisories?offset=0&limit=20&sort=-public_date", output.Links.Last)
		assert.Nil(t, output.Links.Next)
		assert.Nil(t, output.Links.Previous)

		// meta
		assert.Equal(t, 0, output.Meta.Offset)
		assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
		assert.Equal(t, 1, output.Meta.TotalItems)
	})
}

func TestAdvisoriesSearchFilter(t *testing.T) {
	testAdvisoriesOk(t, "GET", "/?search=adv-3&filter[advisory_type]=1", func(output AdvisoriesResponse) {
		assert.Equal(t, 0, len(output.Data))
	})
}
