package controllers

import (
	"app/base/core"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func doTestView(t *testing.T, handler gin.HandlerFunc, limit, offset *int, checker func(w *httptest.ResponseRecorder)) {
	core.SetupTest(t)
	body := SystemsAdvisoriesRequest{
		Systems:    []SystemID{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		Advisories: []AdvisoryName{"RH-1", "RH-2"},
		Limit:      limit,
		Offset:     offset,
	}
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/", bytes.NewBuffer(bodyJSON), "", handler, 1, "POST", "/")
	checker(w)
}

func TestSystemsAdvisoriesView(t *testing.T) {
	doTestView(t, PostSystemsAdvisories, nil, nil, func(w *httptest.ResponseRecorder) {
		var output SystemsAdvisoriesResponse
		CheckResponse(t, w, http.StatusOK, &output)
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], AdvisoryName("RH-1"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], AdvisoryName("RH-2"))
		assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000002"][0], AdvisoryName("RH-1"))
	})
}

func TestAdvisoriesSystemsView(t *testing.T) {
	doTestView(t, PostAdvisoriesSystems, nil, nil, func(w *httptest.ResponseRecorder) {
		var output AdvisoriesSystemsResponse
		CheckResponse(t, w, http.StatusOK, &output)
		assert.Equal(t, output.Data["RH-1"][0], SystemID("00000000-0000-0000-0000-000000000001"))
		assert.Equal(t, output.Data["RH-1"][1], SystemID("00000000-0000-0000-0000-000000000002"))
		assert.Equal(t, output.Data["RH-2"][0], SystemID("00000000-0000-0000-0000-000000000001"))
	})
}

func TestSystemAdvisoriesViewOffsetLimit(t *testing.T) {
	limit := 3
	offset := 0
	doTestView(t, PostSystemsAdvisories, &limit, &offset, func(w *httptest.ResponseRecorder) {
		var output SystemsAdvisoriesResponse
		CheckResponse(t, w, http.StatusOK, &output)
		assert.Equal(t, 2, len(output.Data))
		_, has := output.Data["00000000-0000-0000-0000-000000000001"]
		assert.True(t, has)
	})
}

func TestSystemAdvisoriesViewOffsetOverflow(t *testing.T) {
	limit := 1
	offset := 100
	doTestView(t, PostSystemsAdvisories, &limit, &offset, func(w *httptest.ResponseRecorder) {
		var errResp utils.ErrorResponse
		CheckResponse(t, w, http.StatusBadRequest, &errResp)
		assert.Equal(t, InvalidOffsetMsg, errResp.Error)
	})
}

func TestSystemAdvisoriesViewWrongOffset(t *testing.T) {
	offset := 1000
	doTestView(t, PostSystemsAdvisories, nil, &offset, func(w *httptest.ResponseRecorder) {
		var errResp utils.ErrorResponse
		CheckResponse(t, w, http.StatusBadRequest, &errResp)
		assert.Equal(t, InvalidOffsetMsg, errResp.Error)
	})
}

func TestAvisorySystemsViewOffsetLimit(t *testing.T) {
	limit := 3
	offset := 0
	doTestView(t, PostAdvisoriesSystems, &limit, &offset, func(w *httptest.ResponseRecorder) {
		var output AdvisoriesSystemsResponse
		CheckResponse(t, w, http.StatusOK, &output)
		assert.Equal(t, 2, len(output.Data))
		_, has := output.Data["RH-1"]
		assert.True(t, has)
	})
}

func TestAvisorySystemsViewOffsetOverflow(t *testing.T) {
	limit := 1
	offset := 100
	doTestView(t, PostAdvisoriesSystems, &limit, &offset, func(w *httptest.ResponseRecorder) {
		var errResp utils.ErrorResponse
		CheckResponse(t, w, http.StatusBadRequest, &errResp)
		assert.Equal(t, InvalidOffsetMsg, errResp.Error)
	})
}

func TestAvisorySystemsViewWrongOffset(t *testing.T) {
	offset := 1000
	doTestView(t, PostAdvisoriesSystems, nil, &offset, func(w *httptest.ResponseRecorder) {
		var errResp utils.ErrorResponse
		CheckResponse(t, w, http.StatusBadRequest, &errResp)
		assert.Equal(t, InvalidOffsetMsg, errResp.Error)
	})
}
