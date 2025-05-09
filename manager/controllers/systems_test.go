package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sapABCFilter = "?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids]=ABC"

func TestSystemsDefault(t *testing.T) {
	output := testSystems(t, ``, 1)

	// data
	assert.Equal(t, 10, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "2020-09-22 16:00:00 +0000 UTC", output.Data[0].Attributes.LastUpload.String())
	assert.Equal(t, 1, output.Data[0].Attributes.RheaCount)
	assert.Equal(t, 2, output.Data[0].Attributes.RhbaCount)
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
	assert.Equal(t, 0, output.Data[0].Attributes.PackagesInstalled)
	assert.Equal(t, "RHEL 8.10", output.Data[0].Attributes.OS)
	assert.Equal(t, "8.10", output.Data[0].Attributes.Rhsm)
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output.Data[0].Attributes.StaleTimestamp.String())
	assert.Equal(t, "2018-09-02 16:00:00 +0000 UTC", output.Data[0].Attributes.StaleWarningTimestamp.String())
	assert.Equal(t, "2018-09-09 16:00:00 +0000 UTC", output.Data[0].Attributes.CulledTimestamp.String())
	assert.Equal(t, "2018-08-26 16:00:00 +0000 UTC", output.Data[0].Attributes.Created.String())
	assert.Equal(t, SystemTagsList{{"k1", "ns1", "val1"}, {"k2", "ns1", "val2"}}, output.Data[0].Attributes.Tags)
	assert.Equal(t, "baseline_1-1", output.Data[0].Attributes.BaselineName)
	assert.Equal(t, int64(1), output.Data[0].Attributes.BaselineID)
	assert.False(t, output.Data[0].Attributes.SatelliteManaged)
	assert.False(t, output.Data[0].Attributes.BuiltPkgcache)
	assert.Equal(t, "x86_64", output.Data[0].Attributes.Arch)
	assert.Equal(t, "temp1-1", output.Data[0].Attributes.TemplateName)
	assert.Equal(t, "99900000-0000-0000-0000-000000000001", output.Data[0].Attributes.TemplateUUID)

	// links
	assert.Equal(t, "/?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.First)
	assert.Equal(t, "/?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// test meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 10, output.Meta.TotalItems)
	assert.Equal(t, 9, output.Meta.SubTotals["patched"])
	assert.Equal(t, 1, output.Meta.SubTotals["unpatched"])
	assert.Equal(t, 0, output.Meta.SubTotals["stale"])
}

func TestSystemsOffset(t *testing.T) {
	output := testSystems(t, `?offset=0&limit=4`, 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 10, output.Meta.TotalItems)
}

func TestSystemsOffsetLimit(t *testing.T) {
	output := testSystems(t, `?offset=4&limit=4`, 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 4, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 10, output.Meta.TotalItems)
}

func TestSystemsWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "", "?offset=13&limit=4", SystemsListHandler)
}

func TestSystemsWrongSort(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "?sort=unknown_key")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid sort field: unknown_key", errResp.Error)
}

func TestSystemsSearch(t *testing.T) {
	output := testSystems(t, "?search=001", 1)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)
}

func TestSystemsTags(t *testing.T) {
	output := testSystems(t, "?tags=ns1/k2=val2", 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsTagsMultiple(t *testing.T) {
	output := testSystems(t, "?tags=ns1/k3=val4&tags=ns1/k1=val1", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsUnknown(t *testing.T) {
	output := testSystems(t, "?tags=ns1/k3=val4&tags=ns1/k1=unk", 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsNoVal(t *testing.T) {
	output := testSystems(t, "?tags=ns1/k3=val4&tags=ns1/k1", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsInvalid(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "?tags=ns1/k3=val4&tags=invalidTag")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsTagsEscaping1(t *testing.T) {
	output := testSystems(t, `?tags=ns1/k3=val4&tags="ns/key=quote"`, 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsEscaping2(t *testing.T) {
	output := testSystems(t, `?tags=ns1/k3=val4&tags='ns/key=singlequote'`, 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsEscaping3(t *testing.T) {
	output := testSystems(t, `?tags=ns1/k3=val4&tags='ns/key=inside""quote'`, 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsEscaping4(t *testing.T) {
	output := testSystems(t, `?tags=ns1/k3=val4&tags=ne/key="{{malformed json}}"`, 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsWorkloads1(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids]=ABC", 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads2(t *testing.T) {
	output := testSystems(t, sapABCFilter, 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads3(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_system]=false", 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsWorkloadEscaping1(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_sids]='singlequote'", 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsWorkloadEscaping2(t *testing.T) {
	output := testSystems(t, `?filter[system_profile][sap_sids]="{{malformed json}}"`, 1)
	assert.Equal(t, 0, len(output.Data))
}
func TestSystemsPackagesCount(t *testing.T) {
	output := testSystems(t, "?sort=-packages_installed,id", 3)
	assert.Equal(t, 5, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000015", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000015", output.Data[0].Attributes.DisplayName)
	assert.Equal(t, 3, output.Data[0].Attributes.PackagesInstalled)
}

func TestSystemsFilterAdvCount1(t *testing.T) {
	output := testSystems(t, "?filter[rhba_count]=2", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.RhbaCount)
}

func TestSystemsFilterAdvCount2(t *testing.T) {
	output := testSystems(t, "?filter[rhea_count]=1", 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 1, output.Data[0].Attributes.RheaCount)
}

func TestSystemsFilterAdvCount3(t *testing.T) {
	output := testSystems(t, "?filter[rhsa_count]=2", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
}

func TestSystemsFilterAdvCount4(t *testing.T) {
	output := testSystems(t, "?filter[other_count]=4", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 4, output.Data[0].Attributes.OtherCount)
}

func TestSystemsFilterBaseline(t *testing.T) {
	output := testSystems(t, "?filter[baseline_name]=baseline_1-1", 1)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000018", output.Data[1].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[2].ID)
}

func TestSystemsFilterNotExisting(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "?filter[not-existing]=1")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "cannot parse inventory filters: Invalid filter field: not-existing", errResp.Error)
}

func TestSystemsFilterOS(t *testing.T) {
	output := testSystems(t, `?filter[os]=in:RHEL 8.1,RHEL 7.3&sort=os`, 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, "RHEL 7.3", output.Data[0].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[1].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[2].Attributes.OS)
}

func TestSystemsFilterPartialOS(t *testing.T) {
	output := testSystems(t, "?filter[osname]=RHEL&filter[osmajor]=8&filter[osminor]=1", 1)
	assert.Equal(t, 3, len(output.Data))
	for _, d := range output.Data {
		assert.Equal(t, "RHEL 8.1", d.Attributes.OS)
	}
}

func TestSystemsOrderOS(t *testing.T) {
	output := testSystems(t, `?sort=os`, 1)
	assert.Equal(t, "RHEL 7.3", output.Data[0].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[1].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[2].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[3].Attributes.OS)
	assert.Equal(t, "RHEL 8.2", output.Data[4].Attributes.OS)
	assert.Equal(t, "RHEL 8.2", output.Data[5].Attributes.OS)
	assert.Equal(t, "RHEL 8.3", output.Data[6].Attributes.OS)
	assert.Equal(t, "RHEL 8.3", output.Data[7].Attributes.OS)
	assert.Equal(t, "RHEL 8.10", output.Data[8].Attributes.OS) // yes, we should be robust against this
	assert.Equal(t, "RHEL 8.x", output.Data[9].Attributes.OS)  // yes, we should be robust against this
}

func TestSystemsFilterArch(t *testing.T) {
	output := testSystems(t, `?filter[arch]=x86_64`, 1)
	assert.Equal(t, 9, len(output.Data))
	for _, d := range output.Data {
		assert.Equal(t, "x86_64", d.Attributes.Arch)
	}
}

func TestSystemsFilterTemplateName(t *testing.T) {
	output := testSystems(t, `?filter[template_name]=temp1-1`, 1)
	assert.Equal(t, 2, len(output.Data))
	for _, d := range output.Data {
		assert.Equal(t, "temp1-1", d.Attributes.TemplateName)
	}
}

func TestSystemsFilterTemplateUUID(t *testing.T) {
	output := testSystems(t, `?filter[template_uuid]=99900000-0000-0000-0000-000000000001`, 1)
	assert.Equal(t, 2, len(output.Data))
	for _, d := range output.Data {
		assert.Equal(t, "99900000-0000-0000-0000-000000000001", d.Attributes.TemplateUUID)
	}
}

func testSystems(t *testing.T, queryString string, account int) SystemsResponse {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/", "", queryString, nil, "", SystemsListHandler, account)

	var output SystemsResponse
	CheckResponse(t, w, http.StatusOK, &output)
	return output
}

func testSystemsError(t *testing.T, queryString string) (int, utils.ErrorResponse) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", "/", "", queryString, nil, "", SystemsListHandler)

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	return w.Code, errResp
}

func TestSystemsTagsInMetadata(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/", "", "?tags=ns1/k3=val4&tags=ns1/k1=val1", nil, "",
		SystemsListHandler, 3)

	var output SystemsResponse
	ParseResponseBody(t, w.Body.Bytes(), &output)

	testMap := map[string]FilterData{
		"ns1/k1": {Operator: "eq", Values: []string{"val1"}},
		"ns1/k3": {Operator: "eq", Values: []string{"val4"}},
		"stale":  {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta1(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_sids]=ABC", 1)
	testMap := map[string]FilterData{
		"system_profile][sap_sids": {Operator: "eq", Values: []string{"ABC"}},
		"stale":                    {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta2(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_sids]=ABC", 1)
	testMap := map[string]FilterData{
		"system_profile][sap_sids": {Operator: "eq", Values: []string{"ABC"}},
		"stale":                    {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta3(t *testing.T) {
	output := testSystems(t, sapABCFilter, 1)
	testMap := map[string]FilterData{
		"system_profile][sap_system": {Operator: "eq", Values: []string{"true"}},
		"system_profile][sap_sids":   {Operator: "eq", Values: []string{"ABC"}},
		"stale":                      {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta4(t *testing.T) {
	output := testSystems(t, "?filter[system_profile][sap_sids]=ABC&filter[system_profile][sap_sids]=GHI", 1)
	testMap := map[string]FilterData{
		"system_profile][sap_sids": {Operator: "eq", Values: []string{"GHI", "ABC"}},
		"stale":                    {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta(t *testing.T) {
	output := testSystems(t, `?filter[system_profile][ansible][controller_version]=1.0`, 1)
	testMap := map[string]FilterData{
		"system_profile][ansible][controller_version": {Operator: "eq", Values: []string{"1.0"}},
		"stale": {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta2(t *testing.T) {
	output := testSystems(t, `?filter[system_profile][ansible]=not_nil`, 1)
	testMap := map[string]FilterData{
		"system_profile][ansible": {Operator: "eq", Values: []string{"not_nil"}},
		"stale":                   {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta3(t *testing.T) {
	const (
		ID         = "00000000-0000-0000-0000-000000000007"
		totalItems = 2
	)
	output := testSystems(t, `?filter[system_profile][ansible][controller_version]=1.0`, 1)
	assert.Equal(t, ID, output.Data[0].ID)
	assert.Equal(t, totalItems, output.Meta.TotalItems)
}

func TestMSSQLSystemMeta(t *testing.T) {
	output := testSystems(t, `?filter[system_profile][mssql][version]=15.3.0`, 1)
	testMap := map[string]FilterData{
		"system_profile][mssql][version": {Operator: "eq", Values: []string{"15.3.0"}},
		"stale":                          {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestMSSQLSystemMeta2(t *testing.T) {
	output := testSystems(t, `?filter[system_profile][mssql]=not_nil`, 1)
	testMap := map[string]FilterData{
		"system_profile][mssql": {Operator: "eq", Values: []string{"not_nil"}},
		"stale":                 {Operator: "eq", Values: []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestMSSQLSystemMeta3(t *testing.T) {
	const (
		ID         = "00000000-0000-0000-0000-000000000006"
		totalItems = 2
	)
	output := testSystems(t, `?filter[system_profile][mssql][version]=15.3.0`, 1)
	assert.Equal(t, ID, output.Data[0].ID)
	assert.Equal(t, totalItems, output.Meta.TotalItems)
}
