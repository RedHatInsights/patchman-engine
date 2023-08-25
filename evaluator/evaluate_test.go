package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var systemID = int64(12)
var rhAccountID = 3

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/evaluator_common.env", "conf/evaluator_upload.env")
}

// nolint: funlen
func TestEvaluate(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	// don't use vmaas-cache since tests here are not using vmaas_json
	// so it will always get the same result from cache for empty vmaas_json
	os.Setenv("ENABLE_VMAAS_CACHE", "false")
	defer os.Setenv("ENABLE_VMAAS_CACHE", "true")

	configure()
	loadCache()
	mockWriter := mqueue.MockKafkaWriter{}
	remediationsPublisher = &mockWriter

	expectedAddedAdvisories := []string{"RH-1", "RH-2"}
	expectedAdvisoryIDs := []int64{1, 2}       // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int64{1, 3, 4}   // old advisories paired with the system
	patchingSystemAdvisoryIDs := []int64{3, 4} // these advisories should be patched for the system
	expectedPackageIDs := []int64{1, 2}
	systemRepoIDs := []int64{1, 2}

	database.DeleteSystemAdvisories(t, systemID, expectedAdvisoryIDs)
	database.DeleteSystemAdvisories(t, systemID, patchingSystemAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, patchingSystemAdvisoryIDs)
	database.DeleteSystemPackages(t, systemID, expectedPackageIDs...)
	database.DeleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CreateSystemAdvisories(t, rhAccountID, systemID, oldSystemAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CreateSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CheckCachesValid(t)

	// do evaluate the system
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs:  []string{"00000000-0000-0000-0000-000000000012", "00000000-0000-0000-0000-000000000011"},
		RequestIDs: []string{"request-1", "request-2"},
		AccountID:  rhAccountID})
	assert.NoError(t, err)

	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckSystemAdvisories(t, systemID, advisoryIDs)
	database.CheckSystemPackages(t, systemID, len(expectedPackageIDs), expectedPackageIDs...)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, false)
	database.CheckCachesValid(t)

	// test evaluation with third party repos
	thirdPartySystemRepoIDs := []int64{1, 2, 4}
	database.DeleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CreateSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)
	err = evaluateHandler(mqueue.PlatformEvent{
		SystemIDs:  []string{"00000000-0000-0000-0000-000000000012"},
		RequestIDs: []string{"request-1"},
		AccountID:  rhAccountID})
	assert.NoError(t, err)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, true)

	database.DeleteSystemAdvisories(t, systemID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)
	database.DeleteSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)

	assert.Equal(t, 2, len(mockWriter.Messages))
}

func TestEvaluateYum(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()

	const (
		ID    = "00000000-0000-0000-0000-000000000015"
		sysID = 15
	)

	mockWriter := mqueue.MockKafkaWriter{}
	remediationsPublisher = &mockWriter
	evalLabel = recalcLabel

	expectedAddedAdvisories := []string{"RH-1", "RH-2", "RHSA-2021:3801"}
	expectedAdvisoryIDs := []int64{1, 2, 14} // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int64{1, 2}    // old advisories paired with the system
	expectedPackageIDs := []int64{1, 2}

	database.DeleteSystemAdvisories(t, sysID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.CreateSystemAdvisories(t, rhAccountID, sysID, oldSystemAdvisoryIDs)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CheckCachesValid(t)

	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs: []string{ID},
		AccountID: rhAccountID})
	assert.NoError(t, err)

	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckSystemJustEvaluated(t, ID, 3, 1, 1,
		1, 2, 2, false)

	database.DeleteSystemPackages(t, sysID, expectedPackageIDs...)
	database.DeleteSystemAdvisories(t, sysID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)

	assert.Equal(t, 1, len(mockWriter.Messages))
}

func TestEvaluatePruneUpdates(t *testing.T) {
	assert.NoError(t, os.Setenv("PRUNE_UPDATES_LATEST_ONLY", "true"))
	defer os.Setenv("PRUNE_UPDATES_LATEST_ONLY", "false")

	TestEvaluate(t)
	count := database.CheckSystemUpdatesCount(t, rhAccountID, systemID)
	for _, c := range count {
		assert.LessOrEqual(t, c, 1, "All packages should only have single update stored")
	}
}

func TestEvaluateDontPruneUpdates(t *testing.T) {
	assert.NoError(t, os.Setenv("PRUNE_UPDATES_LATEST_ONLY", "false"))
	TestEvaluate(t)
	count := database.CheckSystemUpdatesCount(t, rhAccountID, systemID)
	oneGreater := false
	for _, c := range count {
		oneGreater = oneGreater || (c > 1)
	}
	assert.True(t, oneGreater,
		"At least one package should have multiple updates (OR we have package pruning enabled)")
}

func TestRun(t *testing.T) {
	configure()
	var nReaders int32
	wg := sync.WaitGroup{}
	run(&wg, mqueue.CreateCountedMockReader(&nReaders))
	wg.Wait()
	assert.Equal(t, 8, int(nReaders)) // 8 - is default
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
	ctx := context.Background()
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

	system := &models.SystemPlatform{YumUpdates: data}
	updates, err := tryGetYumUpdates(system)
	updateList := updates.GetUpdateList()["kernel-2.6.32-696.20.1.el6.x86_64"]
	assert.Nil(t, err)
	assert.NotNil(t, updates)
	assert.Equal(t, 2, len(updateList.GetAvailableUpdates()))
}

// nolint:funlen
func TestSatelliteSystemAdvisories(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	ogYumUpdatesEval := enableYumUpdatesEval
	enableYumUpdatesEval = true
	defer func() { enableYumUpdatesEval = ogYumUpdatesEval }()

	configure()
	loadCache()
	mockWriter := mqueue.MockKafkaWriter{}
	remediationsPublisher = &mockWriter

	vmaasJSON := `
	{
		"package_list": [
			"git-2.30.1-1.el8_8.x86_64",
			"sqlite-3.21.0-1.el8_6.x86_64"
		],
		"repository_list": [
			"rhel-8-for-x86_64-appstream-rpms"
		],
		"releasever": "8",
		"basearch": "x86_64",
		"latest_only": true
	}
	`
	// this satellite system has 2 git and 1 sqlite advisories reported by vmaas (APPLICABLE)
	vmaasDataResp := `
	{
		"update_list": {
			"git-2.30.1-1.el8_8.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2023:3246",
						"basearch": "x86_64",
						"releasever": "8",
						"repository": "rhel-8-for-x86_64-appstream-rpms",
						"package": "git-2.39.3-1.el8_8.x86_64"
					},
					{
						"erratum": "RHSA-2023:3240",
						"basearch": "x86_64",
						"releasever": "8",
						"repository": "rhel-8-for-x86_64-appstream-rpms",
						"package": "git-2.39.4-1.el8_8.x86_64"
					}
				]
			},
			"sqlite-3.21.0-1.el8_6.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2022:7100",
						"basearch": "x86_64",
						"releasever": "8",
						"repository": "rhel-8-for-x86_64-appstream-rpms",
						"package": "sqlite-3.26.0-16.el8_6.x86_64"
					}
				]
			}
		}
	}
	`

	var vmaasData vmaas.UpdatesV3Response
	err := json.Unmarshal([]byte(vmaasDataResp), &vmaasData)
	assert.Nil(t, err)

	// lets add the checksum to the cache, so we do not actually call vmaas
	vmaasJSONChecksum := "1337"
	memoryVmaasCache.Add(&vmaasJSONChecksum, &vmaasData)

	// this satellite system has 1 git installable advisory which is the same as the applicable one from vmaas
	// and 1 sqlite different installable advisory
	yumUpdatesRaw := []byte(`
		{
			"update_list": {
				"git-2.30.1-1.el8_8.x86_64": {
					"available_updates": [
						{
							"erratum": "RHSA-2023:3246",
							"basearch": "x86_64",
							"releasever": "8",
							"repository": "rhel-8-for-x86_64-appstream-rpms",
							"package": "git-2.39.3-1.el8_8.x86_64"
						}
					]
				},
				"sqlite-3.21.0-1.el8_6.x86_64": {
					"available_updates": [
						{
							"erratum": "RHSA-2022:7108",
							"basearch": "x86_64",
							"releasever": "8",
							"repository": "rhel-8-for-x86_64-appstream-rpms",
							"package": "sqlite-3.26.0-16.el8_6.x86_64"
						}
					]
				}
			}
		}
		`)

	system := models.SystemPlatform{
		InventoryID:      "99999999-0000-0000-0000-000000000015",
		JSONChecksum:     &vmaasJSONChecksum,
		VmaasJSON:        &vmaasJSON,
		YumUpdates:       yumUpdatesRaw,
		DisplayName:      "satellite_system_test1",
		RhAccountID:      1,
		BuiltPkgcache:    true,
		SatelliteManaged: true,
	}
	tx := database.Db.Create(&system)
	assert.Nil(t, tx.Error)

	result, err := getUpdatesData(context.Background(), database.Db, &system)
	assert.Nil(t, err)

	// result should have 2 git advisories,    1 is installable (taken from yum updates and vmaas, merged)
	//                                         1 is applicable  (taken from vmaas)
	//                    2 sqlite advisories, 1 is installable (taken from yum updates)
	//                                         1 is applicable  (taken from vmaas)
	var installableCnt, applicableCnt int
	for _, updates := range result.GetUpdateList() {
		for _, update := range updates.GetAvailableUpdates() {
			if update.StatusID == INSTALLABLE {
				installableCnt++
			} else if update.StatusID == APPLICABLE {
				applicableCnt++
			}
		}
	}
	assert.Equal(t, 2, installableCnt)
	assert.Equal(t, 2, applicableCnt)

	database.Db.Delete(system)
}

func TestCallVmaas400(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	configure()
	req := vmaas.UpdatesV3Request{TestReturnStatus: 400}
	_, err := callVMaas(context.Background(), &req)
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
	reqJSON, _ := json.Marshal(req)
	reqString := string(reqJSON)
	sp := models.SystemPlatform{VmaasJSON: &reqString}
	res, err := getUpdatesData(context.Background(), database.Db, &sp)
	// response and error should be nil, system is skipped due to VMaaS 400
	assert.Nil(t, err)
	assert.Nil(t, res)
}
