package controllers

import (
	"app/base/core"
	"app/base/utils"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func doTestView(t *testing.T, handler gin.HandlerFunc, q string, limit, offset *int) *httptest.ResponseRecorder {
	core.SetupTest(t)
	body := SystemsAdvisoriesRequest{
		Systems:    []SystemID{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		Advisories: []AdvisoryName{"RH-1", "RH-3"},
		Limit:      limit,
		Offset:     offset,
	}
	bodyJSON, err := sonic.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/", "", q, bytes.NewBuffer(bodyJSON), "", handler, 1)
	return w
}

func TestSystemsAdvisoriesView(t *testing.T) {
	w := doTestView(t, PostSystemsAdvisories, "", nil, nil)
	var output SystemsAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], AdvisoryName("RH-1"))
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], AdvisoryName("RH-3"))
	assert.Equal(t, 0, len(output.Data["00000000-0000-0000-0000-000000000002"]))
}

func TestAdvisoriesSystemsView(t *testing.T) {
	w := doTestView(t, PostAdvisoriesSystems, "", nil, nil)
	var output AdvisoriesSystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, output.Data["RH-1"][0], SystemID("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, output.Data["RH-3"][0], SystemID("00000000-0000-0000-0000-000000000001"))
}

func TestSystemsAdvisoriesViewTags(t *testing.T) {
	w := doTestView(t, PostSystemsAdvisories, "?filter[system_profile][sap_sids]=DEF", nil, nil)
	var output SystemsAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], AdvisoryName("RH-1"))
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], AdvisoryName("RH-3"))
}

func TestAdvisoriesSystemsViewTags(t *testing.T) {
	w := doTestView(t, PostAdvisoriesSystems, "?filter[system_profile][sap_sids]=DEF", nil, nil)
	var output AdvisoriesSystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, output.Data["RH-1"][0], SystemID("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, output.Data["RH-3"][0], SystemID("00000000-0000-0000-0000-000000000001"))
}
func TestSystemAdvisoriesViewOffsetLimit(t *testing.T) {
	limit := 3
	offset := 0
	w := doTestView(t, PostSystemsAdvisories, "", &limit, &offset)
	var output SystemsAdvisoriesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, limit, output.Meta.Limit)
	assert.Equal(t, offset, output.Meta.Offset)
	_, has := output.Data["00000000-0000-0000-0000-000000000001"]
	assert.True(t, has)
}

func TestSystemAdvisoriesViewOffsetOverflow(t *testing.T) {
	limit := 1
	offset := 100
	w := doTestView(t, PostSystemsAdvisories, "", &limit, &offset)
	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestSystemAdvisoriesViewWrongOffset(t *testing.T) {
	offset := 1000
	w := doTestView(t, PostSystemsAdvisories, "", nil, &offset)
	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestAvisorySystemsViewOffsetLimit(t *testing.T) {
	limit := 3
	offset := 0
	w := doTestView(t, PostAdvisoriesSystems, "", &limit, &offset)
	var output AdvisoriesSystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, limit, output.Meta.Limit)
	assert.Equal(t, offset, output.Meta.Offset)
	_, has := output.Data["RH-1"]
	assert.True(t, has)
}

func TestAvisorySystemsViewOffsetOverflow(t *testing.T) {
	limit := 1
	offset := 100
	w := doTestView(t, PostAdvisoriesSystems, "", &limit, &offset)
	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestAvisorySystemsViewWrongOffset(t *testing.T) {
	offset := 1000
	w := doTestView(t, PostAdvisoriesSystems, "", nil, &offset)
	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}
