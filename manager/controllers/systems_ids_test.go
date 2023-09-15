package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemsIDsDefault(t *testing.T) {
	output := testSystemsIDs(t, `/`, 1)

	// data
	assert.Equal(t, 8, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
}

func TestSystemsIDsOffset(t *testing.T) {
	output := testSystemsIDs(t, `/?offset=0&limit=4`, 1)
	assert.Equal(t, 4, len(output.IDs))
}

func TestSystemsIDsOffsetLimit(t *testing.T) {
	output := testSystemsIDs(t, `/?offset=4&limit=4`, 1)
	assert.Equal(t, 4, len(output.IDs))
}

func TestSystemsIDsWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=13&limit=4", SystemsListIDsHandler)
}

func TestSystemsIDsWrongSort(t *testing.T) {
	statusCode, errResp := testSystemsIDsError(t, "/?sort=unknown_key")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid sort field: unknown_key", errResp.Error)
}

func TestSystemsIDsSearch(t *testing.T) {
	output := testSystemsIDs(t, "/?search=001", 1)
	assert.Equal(t, 1, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
}

func TestSystemsIDsTags(t *testing.T) {
	output := testSystemsIDs(t, "/?tags=ns1/k2=val2", 1)
	assert.Equal(t, 2, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
}

func TestSystemsIDsTagsMultiple(t *testing.T) {
	output := testSystemsIDs(t, "/?tags=ns1/k3=val4&tags=ns1/k1=val1", 1)
	assert.Equal(t, 1, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.IDs[0])
}

func TestSystemsIDsTagsUnknown(t *testing.T) {
	output := testSystemsIDs(t, "/?tags=ns1/k3=val4&tags=ns1/k1=unk", 1)
	assert.Equal(t, 0, len(output.IDs))
}

func TestSystemsIDsTagsNoVal(t *testing.T) {
	output := testSystemsIDs(t, "/?tags=ns1/k3=val4&tags=ns1/k1", 1)
	assert.Equal(t, 1, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.IDs[0])
}

func TestSystemsIDsTagsInvalid(t *testing.T) {
	statusCode, errResp := testSystemsIDsError(t, "/?tags=ns1/k3=val4&tags=invalidTag")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsIDsWorkloads1(t *testing.T) {
	url := "/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids]=ABC"
	output := testSystemsIDs(t, url, 1)
	assert.Equal(t, 2, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
}

func TestSystemsIDsWorkloads2(t *testing.T) {
	url := sapABCFilter
	output := testSystemsIDs(t, url, 1)
	assert.Equal(t, 2, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
}

func TestSystemsIDsWorkloads3(t *testing.T) {
	output := testSystemsIDs(t, "/?filter[system_profile][sap_system]=false", 1)
	assert.Equal(t, 0, len(output.IDs))
}

func TestSystemsIDsPackagesCount(t *testing.T) {
	output := testSystemsIDs(t, "/?sort=-packages_installed,id", 3)
	assert.Equal(t, 5, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.IDs[0])
}

func TestSystemsIDsFilterAdvCount1(t *testing.T) {
	outputIDs := testSystemsIDs(t, "/?filter[rhba_count]=2", 1)
	output := testSystems(t, "/?filter[rhba_count]=2", 1)
	assert.Equal(t, 1, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
}

func TestSystemsIDsFilterAdvCount2(t *testing.T) {
	outputIDs := testSystemsIDs(t, "/?filter[rhea_count]=1", 1)
	output := testSystems(t, "/?filter[rhea_count]=1", 1)
	assert.Equal(t, 4, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
}

func TestSystemsIDsFilterAdvCount3(t *testing.T) {
	outputIDs := testSystemsIDs(t, "/?filter[rhsa_count]=2", 1)
	output := testSystems(t, "/?filter[rhsa_count]=2", 1)
	assert.Equal(t, 1, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
}

func TestSystemsIDsFilterAdvCount4(t *testing.T) {
	outputIDs := testSystemsIDs(t, "/?filter[other_count]=4", 1)
	output := testSystems(t, "/?filter[other_count]=4", 1)
	assert.Equal(t, 1, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
}

func TestSystemsIDsFilterBaseline(t *testing.T) {
	output := testSystemsIDs(t, "/?filter[baseline_name]=baseline_1-1", 1)
	assert.Equal(t, 2, len(output.IDs))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.IDs[0])
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.IDs[1])
}

func TestSystemsIDsFilterNotExisting(t *testing.T) {
	statusCode, errResp := testSystemsIDsError(t, "/?filter[not-existing]=1")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "cannot parse inventory filters: Invalid filter field: not-existing", errResp.Error)
}

func TestSystemsIDsFilterPartialOS(t *testing.T) {
	outputIDs := testSystemsIDs(t, "/?filter[osname]=RHEL&filter[osmajor]=8&filter[osminor]=1", 1)
	output := testSystems(t, "/?filter[osname]=RHEL&filter[osmajor]=8&filter[osminor]=1", 1)
	assert.Equal(t, 2, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
	assert.Equal(t, output.Data[1].ID, outputIDs.IDs[1])
}

func TestSystemsIDsFilterOS(t *testing.T) {
	outputIDs := testSystemsIDs(t, `/?filter[os]=in:RHEL 8.1,RHEL 7.3&sort=os`, 1)
	output := testSystems(t, `/?filter[os]=in:RHEL 8.1,RHEL 7.3&sort=os`, 1)
	assert.Equal(t, 3, len(outputIDs.IDs))
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
	assert.Equal(t, output.Data[1].ID, outputIDs.IDs[1])
	assert.Equal(t, output.Data[2].ID, outputIDs.IDs[2])
}

func TestSystemsIDsOrderOS(t *testing.T) {
	output := testSystems(t, `/?sort=os`, 1)
	outputIDs := testSystemsIDs(t, `/?sort=os`, 1)
	assert.Equal(t, output.Data[0].ID, outputIDs.IDs[0])
	assert.Equal(t, output.Data[1].ID, outputIDs.IDs[1])
	assert.Equal(t, output.Data[2].ID, outputIDs.IDs[2])
	assert.Equal(t, output.Data[3].ID, outputIDs.IDs[3])
	assert.Equal(t, output.Data[4].ID, outputIDs.IDs[4])
	assert.Equal(t, output.Data[5].ID, outputIDs.IDs[5])
	assert.Equal(t, output.Data[6].ID, outputIDs.IDs[6])
	assert.Equal(t, output.Data[7].ID, outputIDs.IDs[7])
}

func testSystemsIDs(t *testing.T, url string, account int) IDsResponse {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", url, nil, "", SystemsListIDsHandler, "/", account)

	var output IDsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	return output
}

func testSystemsIDsError(t *testing.T, url string) (int, utils.ErrorResponse) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", url, nil, "", SystemsListIDsHandler, "/")

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	return w.Code, errResp
}
