package controllers

import (
	"app/base/core"
	"app/base/utils"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBaselineSystems(t *testing.T, param, queryString string) BaselineSystemsResponse {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", param, queryString, nil, "",
		BaselineSystemsListHandler)

	var output BaselineSystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)

	return output
}

func TestBaselineSystemsDefault(t *testing.T) {
	output := testBaselineSystems(t, "1", "")

	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "baseline_system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000018", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000018", output.Data[0].Attributes.DisplayName)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[1].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[1].Attributes.DisplayName)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[2].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[2].Attributes.DisplayName)

	// links
	assert.Equal(t, "/1/systems?offset=0&limit=20&sort=-display_name", output.Links.First)
	assert.Equal(t, "/1/systems?offset=0&limit=20&sort=-display_name", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestBaselinesystemsEmpty(t *testing.T) {
	output := testBaselineSystems(t, "3", "")

	assert.Equal(t, 0, len(output.Data))
	// links
	assert.Equal(t, "/3/systems?offset=0&limit=20&sort=-display_name", output.Links.First)
	assert.Equal(t, "/3/systems?offset=0&limit=20&sort=-display_name", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 0, output.Meta.TotalItems)
}

func TestBaselineSystemsOffsetLimit(t *testing.T) {
	output := testBaselineSystems(t, "1", "?offset=0&limit=1")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 1, output.Meta.Limit)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestBaselineSystemsUnlimited(t *testing.T) {
	output := testBaselineSystems(t, "1", "?offset=0&limit=-1")
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, -1, output.Meta.Limit)
	assert.Equal(t, 3, output.Meta.TotalItems)
}

func TestBaselineSystemOffsetOverflow(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "?offset=10&limit=4", nil, "",
		BaselineSystemsListHandler)

	var errResp utils.ErrorResponse
	CheckResponse(t, w, http.StatusBadRequest, &errResp)
	assert.Equal(t, InvalidOffsetMsg, errResp.Error)
}

func TestBaselinesFilterDisplayName(t *testing.T) {
	output := testBaselineSystems(t, "1", "?filter[display_name]=00000000-0000-0000-0000-000000000001")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)
}

func TestBaselinesFilterTag(t *testing.T) {
	output := testBaselineSystems(t, "1", "?tags=ns1/k3=val3")
	assert.Equal(t, 1, len(output.Data))
}

func TestBaselineSystemsWrongSort(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/:baseline_id/systems", "1", "?sort=unknown_key", nil, "",
		BaselineSystemsListHandler)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBaselineSystemsSearch(t *testing.T) {
	output := testBaselineSystems(t, "1", "?search=00000000-0000-0000-0000-000000000001")
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "baseline_system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].InventoryID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)

	// links
	assert.Equal(t, "/1/systems?offset=0&limit=20&sort=-display_name&search=00000000-0000-0000-0000-000000000001",
		output.Links.First)
	assert.Equal(t, "/1/systems?offset=0&limit=20&sort=-display_name&search=00000000-0000-0000-0000-000000000001",
		output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 1, output.Meta.TotalItems)
}
