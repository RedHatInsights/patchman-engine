package evaluator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"net/http"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var rhAccountID = 3
var orgID = "org_3"
var testDBID = int64(12)
var testInventoryID = uuid.MustParse("00000000-0000-0000-0000-000000000012")
var testInventoryID2 = uuid.MustParse("00000000-0000-0000-0000-000000000011")

func TestInit(_ *testing.T) {
	utils.TestLoadEnv("conf/evaluator_common.env", "conf/evaluator_upload.env")
}

func TestSystemIDsAndTraceparents(t *testing.T) {
	id1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	single := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	ids, tps := systemIDsAndTraceparents(mqueue.PlatformEvent{
		SystemIDs:    []uuid.UUID{id1, id2},
		Traceparents: []string{"tp1", "tp2"},
	})
	assert.Equal(t, []uuid.UUID{id1, id2}, ids)
	assert.Equal(t, []string{"tp1", "tp2"}, tps)

	ids, tps = systemIDsAndTraceparents(mqueue.PlatformEvent{
		ID:           single,
		Traceparents: []string{"tp-single"},
	})
	assert.Equal(t, []uuid.UUID{single}, ids)
	assert.Equal(t, []string{"tp-single"}, tps)

	assert.Equal(t, "r1", firstRequestID(mqueue.PlatformEvent{RequestIDs: []string{"r1", "r2"}}))
	assert.Equal(t, "", firstRequestID(mqueue.PlatformEvent{}))
}

// nolint: funlen
func TestEvaluate(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()
	mockWriter := mqueue.MockKafkaWriter{}
	remediationsPublisher = &mockWriter

	expectedAddedAdvisories := []string{"RH-1", "RH-2", "RH-100"}
	expectedAdvisoryIDs := []int64{1, 2, 100}  // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int64{1, 3, 4}   // old advisories paired with the system
	patchingSystemAdvisoryIDs := []int64{3, 4} // these advisories should be patched for the system
	expectedPackageIDs := []int64{1, 2}
	systemRepoIDs := []int64{1, 2}

	database.DeleteSystemAdvisories(t, testDBID, expectedAdvisoryIDs)
	database.DeleteSystemAdvisories(t, testDBID, patchingSystemAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, patchingSystemAdvisoryIDs)
	database.DeleteSystemPackages(t, rhAccountID, testDBID, expectedPackageIDs...)
	database.DeleteSystemRepos(t, rhAccountID, testDBID, systemRepoIDs)
	database.CreateSystemAdvisories(t, rhAccountID, testDBID, oldSystemAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CreateSystemRepos(t, rhAccountID, testDBID, systemRepoIDs)
	database.CheckCachesValid(t)

	// do evaluate the system
	data, err := sonic.Marshal(mqueue.PlatformEvent{
		SystemIDs:  []uuid.UUID{testInventoryID, testInventoryID2},
		RequestIDs: []string{"request-1", "request-2"},
		OrgID:      &orgID,
		AccountID:  rhAccountID})
	assert.NoError(t, err)
	err = evaluateHandler(base.Context, mqueue.KafkaMessage{Value: data})
	assert.NoError(t, err)

	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckSystemAdvisories(t, testDBID, advisoryIDs)
	database.CheckSystemPackages(t, rhAccountID, testDBID, len(expectedPackageIDs), expectedPackageIDs...)
	database.CheckSystemJustEvaluated(t, testInventoryID, 3, 1, 1, 0,
		3, 1, 1, 0, 2, 2, 2, false)
	database.CheckCachesValid(t)

	// test evaluation with third party repos
	thirdPartySystemRepoIDs := []int64{1, 2, 4}
	database.DeleteSystemRepos(t, rhAccountID, testDBID, systemRepoIDs)
	database.CreateSystemRepos(t, rhAccountID, testDBID, thirdPartySystemRepoIDs)
	data, err = sonic.Marshal(mqueue.PlatformEvent{
		SystemIDs:  []uuid.UUID{testInventoryID},
		RequestIDs: []string{"request-1"},
		OrgID:      &orgID,
		AccountID:  rhAccountID})
	assert.NoError(t, err)
	err = evaluateHandler(base.Context, mqueue.KafkaMessage{Value: data})
	assert.NoError(t, err)
	database.CheckSystemJustEvaluated(t, testInventoryID, 3, 1, 1, 0,
		3, 1, 1, 0, 2, 2, 2, true)

	database.DeleteSystemAdvisories(t, testDBID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)
	database.DeleteSystemRepos(t, rhAccountID, testDBID, thirdPartySystemRepoIDs)

	assert.Equal(t, 2, len(mockWriter.Messages))
}

func TestEvaluateYum(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()

	testDBID := int64(15)
	testInventoryID := uuid.MustParse("00000000-0000-0000-0000-000000000015")

	mockWriter := mqueue.MockKafkaWriter{}
	remediationsPublisher = &mockWriter
	evalLabel = recalcLabel

	expectedAddedAdvisories := []string{"RH-1", "RH-2", "RH-100", "RHSA-2021:3801"}
	expectedAdvisoryIDs := []int64{1, 2, 100, 14} // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int64{1, 2}         // old advisories paired with the system
	expectedPackages := []string{
		"kernel-5.6.13-200.fc31.x86_64", "firefox-76.0.1-1.fc31.x86_64", "suricata-6.0.3-2.fc35.i686",
	}

	database.DeleteSystemAdvisories(t, testDBID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.CreateSystemAdvisories(t, rhAccountID, testDBID, oldSystemAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CheckCachesValid(t)

	data, err := sonic.Marshal(mqueue.PlatformEvent{
		SystemIDs: []uuid.UUID{testInventoryID},
		OrgID:     &orgID,
		AccountID: rhAccountID})
	assert.NoError(t, err)
	err = evaluateHandler(base.Context, mqueue.KafkaMessage{Value: data})
	assert.NoError(t, err)

	expectedPackageIDs := database.GetPackageIDs(expectedPackages...)
	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckSystemPackages(t, rhAccountID, testDBID, len(expectedPackages), expectedPackageIDs...)
	database.CheckSystemJustEvaluated(t, testInventoryID, 4, 1, 1, 1, 4, 1, 1, 1, 3, 3, 3, false)

	database.DeleteSystemPackages(t, rhAccountID, testDBID, expectedPackageIDs...)
	database.DeleteSystemAdvisories(t, testDBID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)

	assert.Equal(t, 1, len(mockWriter.Messages))
}

func TestRun(t *testing.T) {
	configure()
	var nReaders int32
	wg := sync.WaitGroup{}
	run(&wg, mqueue.CreateCountedMockReader(&nReaders))
	wg.Wait()
	assert.Equal(t, consumerCount, int(nReaders)) // 1 - is default
}

func TestVMaaSUpdatesCall(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()

	req := vmaas.UpdatesV3Request{
		OptimisticUpdates: utils.PtrBool(true),
		PackageList:       []string{"curl-7.29.0-51.el7_6.3.x86_64"},
	}

	resp := vmaas.UpdatesV3Response{}
	ctx := base.Context
	httpResp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasUpdatesURL, &req, &resp) // nolint: bodyclose
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, 2, len(resp.GetUpdateList()))
}

func TestGetYumUpdates(t *testing.T) {
	data := []byte(`
	{
		"update_list": {
			"kernel-2.6.32-696.20.1.el6.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2021:3801",
						"basearch": "x86_64",
						"releasever": "6Server",
						"repository": "rhel-6-server-rpms",
						"package": "kernel-0:3.10.0-696.20.1.el6.x86_64"
					},
					{
						"erratum": "RHSA-2021:3801",
						"basearch": "x86_64",
						"releasever": "6Server",
						"repository": "rhel-6-server-rpms",
						"package": "kernel-0:3.18.0-696.20.1.el6.x86_64"
					}
				]
			}
		},
		"basearch": "x86_64",
		"releasever": "6Server"
	}
	`)

	system := &models.SystemPlatformV2{Inventory: models.SystemInventory{YumUpdates: data}}
	updates, err := tryGetYumUpdates(system)
	updateList := updates.GetUpdateList()["kernel-2.6.32-696.20.1.el6.x86_64"]
	assert.Nil(t, err)
	assert.NotNil(t, updates)
	assert.Equal(t, 2, len(updateList.GetAvailableUpdates()))
}

// platform mock /updates returns RH-1 and RH-2 for firefox, and RH-100 for kernel.
const platformPackagesVmaasJSON = `{
	"package_list": [
		"firefox-0:76.0.1-1.fc31.x86_64",
		"kernel-0:5.6.13-200.fc31.x86_64"
	],
	"repository_list": ["repo1"]
}`

// RH-1 matches the mock firefox update, so the merge keeps the yum copy.
// RH-200 is yum-only.
var platformYumUpdates = []byte(`{
	"update_list": {
		"firefox-0:76.0.1-1.fc31.x86_64": {
			"available_updates": [
				{
					"erratum": "RH-1",
					"basearch": "i686",
					"releasever": "ser1",
					"repository": "repo1",
					"package": "firefox-0:77.0.1-1.fc31.x86_64"
				}
			]
		},
		"kernel-0:5.6.13-200.fc31.x86_64": {
			"available_updates": [
				{
					"erratum": "RH-200",
					"basearch": "i686",
					"releasever": "ser1",
					"repository": "repo1",
					"package": "kernel-0:5.10.0-1.fc31.x86_64"
				}
			]
		}
	}
}`)

func updateStatuses(resp *vmaas.UpdatesV3Response) (int, map[string]int) {
	statuses := map[string]int{}
	total := 0
	for _, updates := range resp.GetUpdateList() {
		for _, update := range updates.GetAvailableUpdates() {
			total++
			statuses[update.GetErratum()] = update.StatusID
		}
	}
	return total, statuses
}

func platformUpdatesSystem(inventoryID, displayName string, satellite, bootc bool) *models.SystemPlatformV2 {
	vmaasJSON := platformPackagesVmaasJSON
	return &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			InventoryID:      uuid.MustParse(inventoryID),
			VmaasJSON:        &vmaasJSON,
			YumUpdates:       platformYumUpdates,
			DisplayName:      displayName,
			RhAccountID:      1,
			BuiltPkgcache:    true,
			SatelliteManaged: satellite,
			Bootc:            bootc,
		},
	}
}

func checkPlatformUpdates(t *testing.T, system *models.SystemPlatformV2, expected map[string]int) {
	t.Helper()
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	ogYumUpdatesEval := enableYumUpdatesEval
	enableYumUpdatesEval = true
	t.Cleanup(func() { enableYumUpdatesEval = ogYumUpdatesEval })

	configure()
	loadCache()
	remediationsPublisher = &mqueue.MockKafkaWriter{}

	result, err := getUpdatesData(base.Context, system)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	total, statuses := updateStatuses(result)
	assert.Equal(t, len(expected), total)
	assert.Equal(t, expected, statuses)
}

func TestSatelliteSystemAdvisories(t *testing.T) {
	// VMaaS updates are applicable. Matching yum RH-1 stays installable; RH-200 is yum-only.
	checkPlatformUpdates(t,
		platformUpdatesSystem("99999999-0000-0000-0000-000000000015", "satellite_system_test1", true, false),
		map[string]int{
			"RH-1":   INSTALLABLE,
			"RH-2":   APPLICABLE,
			"RH-100": APPLICABLE,
			"RH-200": INSTALLABLE,
		})
}

func TestBootcSystemAdvisories(t *testing.T) {
	// Image-mode systems cannot be patched via Insights, so every update is applicable.
	checkPlatformUpdates(t,
		platformUpdatesSystem("99999999-0000-0000-0000-000000000016", "bootc_system_test1", false, true),
		map[string]int{
			"RH-1":   APPLICABLE,
			"RH-2":   APPLICABLE,
			"RH-100": APPLICABLE,
			"RH-200": APPLICABLE,
		})
}

func TestCallVmaas400(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	configure()
	req := vmaas.UpdatesV3Request{TestReturnStatus: 400}
	_, err := callVMaas(base.Context, &req)
	assert.ErrorIs(t, err, errVmaasBadRequest)
}

func TestGetUpdatesDataVmaas400(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	configure()
	loadCache()
	req := vmaas.UpdatesV3Request{
		TestReturnStatus: 400, PackageList: []string{"pkg"}, RepositoryList: []string{"repo"},
	}
	reqJSON, _ := sonic.Marshal(req)
	reqString := string(reqJSON)
	sp := &models.SystemPlatformV2{Inventory: models.SystemInventory{VmaasJSON: &reqString}}
	res, err := getUpdatesData(base.Context, sp)
	// response and error should be nil, system is skipped due to VMaaS 400
	assert.Nil(t, err)
	assert.Nil(t, res)
}
