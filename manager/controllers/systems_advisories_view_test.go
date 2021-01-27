package controllers

import (
	"app/base/core"
	"app/base/database"
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

//nolint: gofmt
func TestAdvisoriesSystemsViewRaw(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var data []systemsAdvisoriesDBLoad
	query := database.Db.Raw("SELECT sp.inventory_id as system_id, am.name as advisory_id" +
	" FROM system_advisories sa join system_platform sp on sp.rh_account_id = 1 and sp.id = sa.system_id" +
	" join advisory_metadata am on am.id = sa.advisory_id JOIN inventory.hosts ih ON ih.id = sp.inventory_id" +
	" WHERE sp.rh_account_id = 1 AND (sp.inventory_id in" +
	" ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002'))" +
	" AND am.name in ('RH-1','RH-2') ORDER BY sp.inventory_id, am.id;")
	err := query.Find(&data).Error;

	assert.NoError(t, err)
	assert.Equal(t, data[0].SystemID, SystemID("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, data[0].AdvisoryID, AdvisoryName("RH-1"))
	assert.Equal(t, data[1].SystemID, SystemID("00000000-0000-0000-0000-000000000001"))
	assert.Equal(t, data[1].AdvisoryID, AdvisoryName("RH-2"))
	assert.Equal(t, data[2].SystemID, SystemID("00000000-0000-0000-0000-000000000002"))
	assert.Equal(t, data[2].AdvisoryID, AdvisoryName("RH-1"))
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
