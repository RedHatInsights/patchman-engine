package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

var testDate, _ = time.Parse(time.RFC3339, "2020-01-01T01-01-01")
var systemID = 12
var rhAccountID = 3

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/evaluator_common.env", "conf/evaluator_upload.env")
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

	expectedAddedAdvisories := []string{"RH-1", "RH-2"}
	expectedAdvisoryIDs := []int{1, 2}       // advisories expected to be paired to the system after evaluation
	oldSystemAdvisoryIDs := []int{1, 3, 4}   // old advisories paired with the system
	patchingSystemAdvisoryIDs := []int{3, 4} // these advisories should be patched for the system
	expectedPackageIDs := []int{1, 2}
	systemRepoIDs := []int{1, 2}

	database.DeleteSystemAdvisories(t, systemID, expectedAdvisoryIDs)
	database.DeleteSystemAdvisories(t, systemID, patchingSystemAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, expectedAdvisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, patchingSystemAdvisoryIDs)
	database.DeleteSystemPackages(t, systemID, expectedPackageIDs...)
	database.DeleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CreateSystemAdvisories(t, rhAccountID, systemID, oldSystemAdvisoryIDs, nil)
	database.CreateAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs, 1)
	database.CreateSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CheckCachesValid(t)

	// do evaluate the system
	err := evaluateHandler(mqueue.PlatformEvent{
		SystemIDs: []string{"00000000-0000-0000-0000-000000000012", "00000000-0000-0000-0000-000000000011"},
		AccountID: rhAccountID})
	assert.NoError(t, err)

	advisoryIDs := database.CheckAdvisoriesInDB(t, expectedAddedAdvisories)
	database.CheckSystemAdvisoriesWhenPatched(t, systemID, advisoryIDs, nil)
	database.CheckSystemPackages(t, systemID, len(expectedPackageIDs), expectedPackageIDs...)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, false)
	database.CheckCachesValid(t)

	// test evaluation with third party repos
	thirdPartySystemRepoIDs := []int{1, 2, 4}
	database.DeleteSystemRepos(t, rhAccountID, systemID, systemRepoIDs)
	database.CreateSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)
	err = evaluateHandler(mqueue.PlatformEvent{
		SystemIDs: []string{"00000000-0000-0000-0000-000000000012"},
		AccountID: rhAccountID})
	assert.NoError(t, err)
	database.CheckSystemJustEvaluated(t, "00000000-0000-0000-0000-000000000012", 2, 1, 1,
		0, 2, 2, true)

	database.DeleteSystemAdvisories(t, systemID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, advisoryIDs)
	database.DeleteAdvisoryAccountData(t, rhAccountID, oldSystemAdvisoryIDs)
	database.DeleteSystemRepos(t, rhAccountID, systemID, thirdPartySystemRepoIDs)

	assert.Equal(t, 2, len(mockWriter.Messages))
}

func TestEvaluatePruneUpdates(t *testing.T) {
	assert.NoError(t, os.Setenv("PRUNE_UPDATES_LATEST_ONLY", "true"))
	defer os.Setenv("PRUNE_UPDATES_LATEST_ONLY", "false")

	TestEvaluate(t)
	count := database.CheckSystemUpdatesCount(t, systemID, rhAccountID)
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
	nReaders := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	run(&wg, mqueue.CreateCountedMockReader(&nReaders))
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return 8, nReaders
	})
}

func TestVMaaSUpdatesCall(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()

	req := vmaas.UpdatesV3Request{
		OptimisticUpdates: utils.PtrBool(true),
		PackageList:       []string{"curl-7.29.0-51.el7_6.3.x86_64"},
	}

	resp := vmaas.UpdatesV2Response{}
	ctx := context.Background()
	httpResp, err := vmaasClient.Request(&ctx, vmaasUpdatesURL, &req, &resp) // nolint: bodyclose
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, 2, len(resp.GetUpdateList()))
}
