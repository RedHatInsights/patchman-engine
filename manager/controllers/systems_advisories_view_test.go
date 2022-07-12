package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func doTestView(t *testing.T, handler gin.HandlerFunc, checker func(w *httptest.ResponseRecorder)) {
	SetupTest(t)
	body := SystemsAdvisoriesRequest{
		Systems:    []SystemID{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		Advisories: []AdvisoryName{"RH-1", "RH-2"},
	}
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/", bytes.NewBuffer(bodyJSON), nil, handler, 1, "POST", "/")
	checker(w)
}

func TestSystemsAdvisoriesView(t *testing.T) {
	doTestView(t, PostSystemsAdvisories, func(w *httptest.ResponseRecorder) {
		assert.Equal(t, http.StatusOK, w.Code)
		var output SystemsAdvisoriesResponse
		ParseResponseBody(t, w.Body.Bytes(), &output)
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], AdvisoryName("RH-1"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], AdvisoryName("RH-2"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000002"][0], AdvisoryName("RH-1"))
	})
}

func TestAdvisoriesSystemsView(t *testing.T) {
	doTestView(t, PostAdvisoriesSystems, func(w *httptest.ResponseRecorder) {
		assert.Equal(t, http.StatusOK, w.Code)
		var output AdvisoriesSystemsResponse
		ParseResponseBody(t, w.Body.Bytes(), &output)
		assert.Equal(t, output.Data["RH-1"][0], SystemID("00000000-0000-0000-0000-000000000001"))
		assert.Equal(t, output.Data["RH-1"][1], SystemID("00000000-0000-0000-0000-000000000002"))
		assert.Equal(t, output.Data["RH-2"][0], SystemID("00000000-0000-0000-0000-000000000001"))
	})
}
