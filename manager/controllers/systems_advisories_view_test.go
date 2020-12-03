package controllers

import (
	"app/base/core"
	"app/base/utils"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemsAdvisoriesView(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	body := SystemsAdvisoriesRequest{
		Systems:    []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"},
		Advisories: []string{"RH-1", "RH-2"},
	}
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBuffer(bodyJSON))
	core.InitRouterWithParams(PostSystemsAdvisories, 1, "POST", "/").
		ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output SystemsAdvisoriesResponse
	ParseReponseBody(t, w.Body.Bytes(), &output)
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][0], "RH-1")
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000001"][1], "RH-2")
	assert.Equal(t, output.Data["00000000-0000-0000-0000-000000000002"][0], "RH-1")
}
