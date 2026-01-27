package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func testTemplateSystemsDelete(t *testing.T, body TemplateSystemsUpdateRequest, status int) *httptest.ResponseRecorder {
	bodyJSON, err := sonic.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/systems", "", "", bytes.NewBuffer(bodyJSON), "",
		TemplateSystemsDeleteHandler, templateAccount)

	assert.Equal(t, status, w.Code)
	return w
}

func TestTemplateSystemsDeleteDefault(t *testing.T) {
	core.SetupTest(t)

	database.CreateTemplate(t, templateAccount, templateUUID, templateSystems)
	template2 := "99999999-9999-8888-8888-888888888888"
	templateSystems2 := []string{
		"00000000-0000-0000-0000-000000000005",
	}
	database.CreateTemplate(t, templateAccount, template2, templateSystems2)

	database.CheckTemplateSystems(t, templateAccount, templateUUID, templateSystems)
	database.CheckTemplateSystems(t, templateAccount, template2, templateSystems2)

	req := TemplateSystemsUpdateRequest{
		Systems: append(templateSystems, templateSystems2...),
	}

	testTemplateSystemsDelete(t, req, http.StatusOK)

	database.CheckTemplateSystems(t, templateAccount, templateUUID, []string{})
	database.CheckTemplateSystems(t, templateAccount, template2, []string{})
	database.DeleteTemplate(t, templateAccount, templateUUID)
	database.DeleteTemplate(t, templateAccount, template2)
}

func TestTemplateSystemsDeleteInvalid(t *testing.T) {
	core.SetupTest(t)

	for _, req := range []TemplateSystemsUpdateRequest{
		{},
		{Systems: []string{}}} {
		testTemplateSystemsDelete(t, req, http.StatusBadRequest)
	}

	testTemplateSystemsDelete(t, TemplateSystemsUpdateRequest{
		Systems: []string{"c0ffeec0-ffee-c0ff-eec0-ffeec0ffee00"}}, http.StatusNotFound)
}

func TestTemplateSystemsDeleteTooManySystems(t *testing.T) {
	core.SetupTest(t)

	systems := make([]string, 0, TemplateSystemsUpdateLimit+1)
	for i := 0; i < TemplateSystemsUpdateLimit; i++ {
		systems = append(systems, uuid.NewString())
	}

	database.CreateTemplate(t, templateAccount, templateUUID, systems)
	defer database.DeleteTemplate(t, templateAccount, templateUUID)

	// Add one more system to the template so we can try to delete more than the limit
	additionalSystem := "00000000-0000-0000-0000-000000000004"
	putBody := TemplateSystemsUpdateRequest{
		Systems: []string{additionalSystem},
	}

	putBodyJSON, err := sonic.Marshal(&putBody)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("PUT", templatePath, templateUUID, "", bytes.NewBuffer(putBodyJSON), "",
		TemplateSystemsUpdateHandler, templateAccount)
	assert.Equal(t, http.StatusOK, w.Code)

	systems = append(systems, additionalSystem)

	req := TemplateSystemsUpdateRequest{
		Systems: systems,
	}

	res := testTemplateSystemsDelete(t, req, http.StatusBadRequest)

	var errResp utils.ErrorResponse
	CheckResponse(t, res, http.StatusBadRequest, &errResp)
	assert.Equal(t, fmt.Sprintf("Cannot process more than %d systems at once", TemplateSystemsUpdateLimit), errResp.Error)
}
