package controllers

import (
	"app/base/core"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func doTestView(t *testing.T, handler gin.HandlerFunc, checker func(w *httptest.ResponseRecorder)) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	body := SystemsAdvisoriesRequest{
		Systems:    []SystemID{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		Advisories: []AdvisoryName{"RH-1", "RH-2"},
	}
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBuffer(bodyJSON))

	core.InitRouterWithParams(handler, 1, "POST", "/").
		ServeHTTP(w, req)
	checker(w)
}

func TestSystemsAdvisoriesView(t *testing.T) {
	doTestView(t, PostSystemsAdvisories, func(w *httptest.ResponseRecorder) {
		assert.Equal(t, 200, w.Code)
		var output SystemsAdvisoriesResponse
		ParseReponseBody(t, w.Body.Bytes(), &output)
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], AdvisoryName("RH-1"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], AdvisoryName("RH-2"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000002"][0], AdvisoryName("RH-1"))
	})
}

func TestAdvisoriesSystemsView(t *testing.T) {
	doTestView(t, PostAdvisoriesSystems, func(w *httptest.ResponseRecorder) {
		assert.Equal(t, 200, w.Code)
		var output AdvisoriesSystemsResponse
		ParseReponseBody(t, w.Body.Bytes(), &output)
		assert.Equal(t, output.Data["RH-1"][0], SystemID("00000000-0000-0000-0000-000000000001"))
		assert.Equal(t, output.Data["RH-1"][1], SystemID("00000000-0000-0000-0000-000000000002"))
		assert.Equal(t, output.Data["RH-2"][0], SystemID("00000000-0000-0000-0000-000000000001"))
	})
}
