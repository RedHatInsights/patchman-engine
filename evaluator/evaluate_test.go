package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

var testDate, _ = time.Parse(time.RFC3339, "2020-01-01T01-01-01")

func TestInit(t *testing.T) {
	utils.TestLoadEnv("conf/evaluator_common.env", "conf/evaluator_upload.env")
}

// nolint: funlen
func TestEvaluate(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()

	configure()
	mockWriter := utils.MockKafkaWriter{}
	remediationsPublisher = &mockWriter

	systemID := 12
	rhAccountID := 3
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
	database.DeleteSystemPackages(t, systemID, expectedPackageIDs)
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
	database.CheckSystemPackages(t, systemID, expectedPackageIDs)
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

func TestRun(t *testing.T) {
	nReaders := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	run(&wg, mqueue.CreateCountedMockReader(&nReaders))
	utils.AssertEqualWait(t, 10, func() (exp, act interface{}) {
		return 8, nReaders
	})
}
