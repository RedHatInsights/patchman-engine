package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testTemplates(t *testing.T, url string) TemplatesResponse {
	core.SetupTest(t)
	w := CreateRequest("GET", url, nil, "", TemplatesListHandler)

	var output TemplatesResponse
	CheckResponse(t, w, http.StatusOK, &output)
	return output
}

func testTemplatesError(t *testing.T, url string, expectedStatus int) utils.ErrorResponse {
	core.SetupTest(t)
	w := CreateRequest("GET", url, nil, "", TemplatesListHandler)

	var output utils.ErrorResponse
	CheckResponse(t, w, expectedStatus, &output)
	return output
}

//nolint:dupl
func TestTemplatesDefault(t *testing.T) {
	output := testTemplates(t, "/")

	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "temp1-1", output.Data[0].Attributes.Name)
	assert.Equal(t, 2, output.Data[0].Attributes.Systems)
	assert.Equal(t, "temp2-1", output.Data[1].Attributes.Name)
	assert.Equal(t, 1, output.Data[1].Attributes.Systems)
	assert.Equal(t, "template", output.Data[2].Type)
	assert.Equal(t, "temp3-1", output.Data[2].Attributes.Name)
	assert.Equal(t, "99900000-0000-0000-0000-000000000003", output.Data[2].ID)
	assert.Equal(t, 0, output.Data[2].Attributes.Systems)

	// links
	assert.Equal(t, "/?offset=0&limit=20&sort=name", output.Links.First)
	assert.Equal(t, "/?offset=0&limit=20&sort=name", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestTemplatesOffsetLimit(t *testing.T) {
	output := testTemplates(t, "/?offset=0&limit=2")
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 2, output.Meta.Limit)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestTemplatesOffsetOverflow(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequest("GET", "/?offset=10&limit=4", nil, "", TemplatesListHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestTemplatesFilterTypeID(t *testing.T) {
	output := testTemplates(t, "/?filter[id]=99900000-0000-0000-0000-000000000001")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "99900000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "temp1-1", output.Data[0].Attributes.Name)
}

func TestTemplatesFilterName(t *testing.T) {
	output := testTemplates(t, "/?filter[name]=temp3-1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "99900000-0000-0000-0000-000000000003", output.Data[0].ID)
	assert.Equal(t, "temp3-1", output.Data[0].Attributes.Name)
}

func TestTemplatesFilterSystems(t *testing.T) {
	output := testTemplates(t, "/?filter[systems]=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "temp2-1", output.Data[0].Attributes.Name)
	assert.Equal(t, 1, output.Data[0].Attributes.Systems)
}

func TestTemplatesFilterTags(t *testing.T) {
	output := testTemplates(t, "/?tags=ns1/k3=val4")
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, 0, output.Data[0].Attributes.Systems)
	assert.Equal(t, 1, output.Data[1].Attributes.Systems)
	assert.Equal(t, 0, output.Data[2].Attributes.Systems)
}

func TestTemplatesFilterTagsBadRequest(t *testing.T) {
	output := testTemplatesError(t, "/?tags=invalidTag", http.StatusBadRequest)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), output.Error)
}

func TestTemplatesSort(t *testing.T) {
	output := testTemplates(t, "/?sort=-name")

	assert.Equal(t, 1, len(output.Meta.Sort))
	assert.Equal(t, "temp3-1", output.Data[0].Attributes.Name)
	assert.Equal(t, output.Meta.Sort[0], "-name")
}

func TestTemplatesWrongSort(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequest("GET", "/?sort=unknown_key", nil, "", TemplatesListHandler)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

//nolint:dupl
func TestTemplatesSearch(t *testing.T) {
	output := testTemplates(t, "/?search=temp1-1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "99900000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "template", output.Data[0].Type)
	assert.Equal(t, "temp1-1", output.Data[0].Attributes.Name)
	assert.Equal(t, 2, output.Data[0].Attributes.Systems)

	// links
	assert.Equal(t, "/?offset=0&limit=20&sort=name&search=temp1-1",
		output.Links.First)
	assert.Equal(t, "/?offset=0&limit=20&sort=name&search=temp1-1",
		output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 1, output.Meta.TotalItems)
}

func TestTemplatesSearchFilter(t *testing.T) {
	output := testTemplates(t, "/?search=template_1-1&filter[systems]=5")
	assert.Equal(t, 0, len(output.Data))
}
