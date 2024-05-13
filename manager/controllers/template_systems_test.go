package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testTemplateSystems(t *testing.T, param, queryString string) TemplateSystemsResponse {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:template_id/systems", param, queryString, nil, "",
		TemplateSystemsListHandler)

	var output TemplateSystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)

	return output
}

func TestTemplateSystemsDefault(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001", "")

	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "template_system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[0].Attributes.DisplayName)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[1].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[1].Attributes.DisplayName)

	// links
	link := "/99900000-0000-0000-0000-000000000001/systems?offset=0&limit=20&sort=-display_name"
	assert.Equal(t, link, output.Links.First)
	assert.Equal(t, link, output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestTemplatesystemsEmpty(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000003", "")

	assert.Equal(t, 0, len(output.Data))
	// links
	link := "/99900000-0000-0000-0000-000000000003/systems?offset=0&limit=20&sort=-display_name"
	assert.Equal(t, link, output.Links.First)
	assert.Equal(t, link, output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 0, output.Meta.TotalItems)
}

func TestTemplateSystemsOffsetLimit(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001", "?offset=0&limit=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 1, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestTemplateSystemsUnlimited(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001", "?offset=0&limit=-1")
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, -1, output.Meta.Limit)
	assert.Equal(t, 2, output.Meta.TotalItems)
}

func TestTemplateSystemOffsetOverflow(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:template_id/systems", "99900000-0000-0000-0000-000000000001",
		"?offset=10&limit=4", nil, "", TemplateSystemsListHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestTemplatesFilterDisplayName(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001",
		"?filter[display_name]=00000000-0000-0000-0000-000000000001")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)
}

func TestTemplatesFilterTag(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001", "?tags=ns1/k3=val3")
	assert.Equal(t, 1, len(output.Data))
}

func TestTemplateSystemsWrongSort(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:template_id/systems", "99900000-0000-0000-0000-000000000001",
		"?sort=unknown_key", nil, "", TemplateSystemsListHandler)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTemplateSystemsSearch(t *testing.T) {
	output := testTemplateSystems(t, "99900000-0000-0000-0000-000000000001",
		"?search=00000000-0000-0000-0000-000000000001")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "template_system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)

	// links
	link := "/99900000-0000-0000-0000-000000000001/systems?" +
		"offset=0&limit=20&sort=-display_name&search=00000000-0000-0000-0000-000000000001"
	assert.Equal(t, link, output.Links.First)
	assert.Equal(t, link, output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 1, output.Meta.TotalItems)
}

func TestTemplateSystemsInvalidUUID(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:template_id/systems", "InvalidTemplateUUID",
		"?sort=unknown_key", nil, "", TemplateSystemsListHandler)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, `{"error":"Invalid template uuid: InvalidTemplateUUID"}`, w.Body.String())
}

func TestTemplateSystemsUnknownUUID(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:template_id/systems", "99999999-9999-9999-9999-999999990001",
		"?sort=unknown_key", nil, "", TemplateSystemsListHandler)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, `{"error":"Template not found"}`, w.Body.String())
}
