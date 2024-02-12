package controllers

import (
	"app/base/core"
	"app/base/database"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testTemplateSystemsDelete(t *testing.T, body TemplateSystemsUpdateRequest, status int) {
	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		panic(err)
	}

	w := CreateRequestRouterWithParams("POST", "/systems", "", "", bytes.NewBuffer(bodyJSON), "",
		TemplateSystemsDeleteHandler, templateAccount)

	assert.Equal(t, status, w.Code)
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

	testTemplateSystemsDelete(t, TemplateSystemsUpdateRequest{Systems: []string{"foo"}}, http.StatusNotFound)
}
