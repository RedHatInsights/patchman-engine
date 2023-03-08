package controllers

import (
	"app/base/core"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sapABCFilter = "/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][]=ABC"

func TestSystemsDefault(t *testing.T) {
	output := testSystems(t, `/`, 1)

	// data
	assert.Equal(t, 8, len(output.Data))
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

	// links
	assert.Equal(t, "/?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.First)
	assert.Equal(t, "/?offset=0&limit=20&filter[stale]=eq:false&sort=-last_upload", output.Links.Last)
	assert.Nil(t, output.Links.Next)
	assert.Nil(t, output.Links.Previous)

	// test meta
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, core.DefaultLimit, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
	assert.Equal(t, 8, output.Meta.SubTotals["patched"])
	assert.Equal(t, 0, output.Meta.SubTotals["unpatched"])
	assert.Equal(t, 0, output.Meta.SubTotals["stale"])
}

func TestSystemsOffset(t *testing.T) {
	output := testSystems(t, `/?offset=0&limit=4`, 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 0, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsOffsetLimit(t *testing.T) {
	output := testSystems(t, `/?offset=4&limit=4`, 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 4, output.Meta.Offset)
	assert.Equal(t, 4, output.Meta.Limit)
	assert.Equal(t, 8, output.Meta.TotalItems)
}

func TestSystemsWrongOffset(t *testing.T) {
	doTestWrongOffset(t, "/", "/?offset=13&limit=4", SystemsListHandler)
}

func TestSystemsWrongSort(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "/?sort=unknown_key")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid sort field: unknown_key", errResp.Error)
}

func TestSystemsSearch(t *testing.T) {
	output := testSystems(t, "/?search=001", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].Attributes.DisplayName)
}

func TestSystemsTags(t *testing.T) {
	output := testSystems(t, "/?tags=ns1/k2=val2", 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsTagsMultiple(t *testing.T) {
	output := testSystems(t, "/?tags=ns1/k3=val4&tags=ns1/k1=val1", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsUnknown(t *testing.T) {
	output := testSystems(t, "/?tags=ns1/k3=val4&tags=ns1/k1=unk", 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsTagsNoVal(t *testing.T) {
	output := testSystems(t, "/?tags=ns1/k3=val4&tags=ns1/k1", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000003", output.Data[0].ID)
}

func TestSystemsTagsInvalid(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "/?tags=ns1/k3=val4&tags=invalidTag")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, fmt.Sprintf(InvalidTagMsg, "invalidTag"), errResp.Error)
}

func TestSystemsWorkloads1(t *testing.T) {
	url := "/?filter[system_profile][sap_system]=true&filter[system_profile][sap_sids][in][]=ABC"
	output := testSystems(t, url, 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads2(t *testing.T) {
	url := sapABCFilter
	output := testSystems(t, url, 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
}

func TestSystemsWorkloads3(t *testing.T) {
	output := testSystems(t, "/?filter[system_profile][sap_system]=false", 1)
	assert.Equal(t, 0, len(output.Data))
}

func TestSystemsPackagesCount(t *testing.T) {
	output := testSystems(t, "/?sort=-packages_installed,id", 3)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].ID)
	assert.Equal(t, "system", output.Data[0].Type)
	assert.Equal(t, "00000000-0000-0000-0000-000000000012", output.Data[0].Attributes.DisplayName)
	assert.Equal(t, 2, output.Data[0].Attributes.PackagesInstalled)
}

func TestSystemsFilterAdvCount1(t *testing.T) {
	output := testSystems(t, "/?filter[rhba_count]=2", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.RhbaCount)
}

func TestSystemsFilterAdvCount2(t *testing.T) {
	output := testSystems(t, "/?filter[rhea_count]=1", 1)
	assert.Equal(t, 4, len(output.Data))
	assert.Equal(t, 1, output.Data[0].Attributes.RheaCount)
}

func TestSystemsFilterAdvCount3(t *testing.T) {
	output := testSystems(t, "/?filter[rhsa_count]=2", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 2, output.Data[0].Attributes.RhsaCount)
}

func TestSystemsFilterAdvCount4(t *testing.T) {
	output := testSystems(t, "/?filter[other_count]=4", 1)
	assert.Equal(t, 1, len(output.Data))
	assert.Equal(t, 4, output.Data[0].Attributes.OtherCount)
}

func TestSystemsFilterBaseline(t *testing.T) {
	output := testSystems(t, "/?filter[baseline_name]=baseline_1-1", 1)
	assert.Equal(t, 2, len(output.Data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", output.Data[0].ID)
	assert.Equal(t, "00000000-0000-0000-0000-000000000002", output.Data[1].ID)
}

func TestSystemsFilterNotExisting(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "/?filter[not-existing]=1")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid filter field: not-existing", errResp.Error)
}

func TestSystemsFilterOS(t *testing.T) {
	output := testSystems(t, `/?filter[os]=in:RHEL 8.1,RHEL 7.3&sort=os`, 1)
	assert.Equal(t, 3, len(output.Data))
	assert.Equal(t, "RHEL 7.3", output.Data[0].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[1].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[2].Attributes.OS)
}

func TestSystemsFilterInvalidSyntax(t *testing.T) {
	statusCode, errResp := testSystemsError(t, "/?filter[os][in]=RHEL 8.1,RHEL 7.3")
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, InvalidNestedFilter, errResp.Error)
}

func TestSystemsOrderOS(t *testing.T) {
	output := testSystems(t, `/?sort=os`, 1)
	assert.Equal(t, "RHEL 7.3", output.Data[0].Attributes.OS)
	assert.Equal(t, "RHEL 8.x", output.Data[1].Attributes.OS) // yes, we should be robust against this
	assert.Equal(t, "RHEL 8.1", output.Data[2].Attributes.OS)
	assert.Equal(t, "RHEL 8.1", output.Data[3].Attributes.OS)
	assert.Equal(t, "RHEL 8.2", output.Data[4].Attributes.OS)
	assert.Equal(t, "RHEL 8.3", output.Data[5].Attributes.OS)
	assert.Equal(t, "RHEL 8.3", output.Data[6].Attributes.OS)
	assert.Equal(t, "RHEL 8.10", output.Data[7].Attributes.OS)
}

func testSystems(t *testing.T, url string, account int) SystemsResponseV3 {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", url, nil, "", SystemsListHandler, "/", account,
		core.ContextKV{Key: middlewares.KeyApiver, Value: 3})

	var output SystemsResponseV3
	CheckResponse(t, w, http.StatusOK, &output)
	return output
}

func testSystemsError(t *testing.T, url string) (int, utils.ErrorResponse) {
	core.SetupTest(t)
	w := CreateRequestRouterWithPath("GET", url, nil, "", SystemsListHandler, "/")

	var errResp utils.ErrorResponse
	ParseResponseBody(t, w.Body.Bytes(), &errResp)
	return w.Code, errResp
}

func TestSystemsTagsInMetadata(t *testing.T) {
	core.SetupTest(t)
	w := CreateRequestRouterWithAccount("GET", "/?tags=ns1/k3=val4&tags=ns1/k1=val1", nil, "", SystemsListHandler, "/", 3,
		core.ContextKV{Key: middlewares.KeyApiver, Value: 3})

	var output SystemsResponseV3
	ParseResponseBody(t, w.Body.Bytes(), &output)

	testMap := map[string]FilterData{
		"ns1/k1": {"eq", []string{"val1"}},
		"ns1/k3": {"eq", []string{"val4"}},
		"stale":  {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta1(t *testing.T) {
	url := "/?filter[system_profile][sap_sids][]=ABC"
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"sap_sids": {"eq", []string{`"ABC"`}},
		"stale":    {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta2(t *testing.T) {
	url := "/?filter[system_profile][sap_sids][in][]=ABC"
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"sap_sids": {"in", []string{`"ABC"`}},
		"stale":    {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta3(t *testing.T) {
	url := sapABCFilter
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"sap_system": {"eq", []string{"true"}},
		"sap_sids":   {"eq", []string{`"ABC"`}},
		"stale":      {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestSAPSystemMeta4(t *testing.T) {
	url := "/?filter[system_profile][sap_sids][in]=ABC&filter[system_profile][sap_sids][in]=GHI"
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"sap_sids": {"in", []string{`"ABC"`, `"GHI"`}},
		"stale":    {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta(t *testing.T) {
	url := `/?filter[system_profile][ansible][controller_version]=1.0`
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"ansible->controller_version": {"eq", []string{"1.0"}},
		"stale":                       {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta2(t *testing.T) {
	url := `/?filter[system_profile][ansible]=not_nil`
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"ansible": {"eq", []string{"not_nil"}},
		"stale":   {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestAAPSystemMeta3(t *testing.T) {
	const (
		ID         = "00000000-0000-0000-0000-000000000007"
		totalItems = 1
	)
	url := `/?filter[system_profile][ansible][controller_version]=1.0`
	output := testSystems(t, url, 1)
	assert.Equal(t, ID, output.Data[0].ID)
	assert.Equal(t, totalItems, output.Meta.TotalItems)
}

func TestMSSQLSystemMeta(t *testing.T) {
	url := `/?filter[system_profile][mssql][version]=15.3.0`
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"mssql->version": {"eq", []string{"15.3.0"}},
		"stale":          {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestMSSQLSystemMeta2(t *testing.T) {
	url := `/?filter[system_profile][mssql]=not_nil`
	output := testSystems(t, url, 1)
	testMap := map[string]FilterData{
		"mssql": {"eq", []string{"not_nil"}},
		"stale": {"eq", []string{"false"}},
	}
	assert.Equal(t, testMap, output.Meta.Filter)
}

func TestMSSQLSystemMeta3(t *testing.T) {
	const (
		ID         = "00000000-0000-0000-0000-000000000006"
		totalItems = 1
	)
	url := `/?filter[system_profile][mssql][version]=15.3.0`
	output := testSystems(t, url, 1)
	assert.Equal(t, ID, output.Data[0].ID)
	assert.Equal(t, totalItems, output.Meta.TotalItems)
}
