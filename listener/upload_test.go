package listener

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"errors"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetOrCreateAccount(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)
	accountID2 := getOrCreateTestAccount(t)
	assert.Equal(t, accountID1, accountID2)

	deleteData(t)
}

func createTestInvHost(t *testing.T) *Host {
	correctTimestamp, err := time.Parse(base.Rfc3339NoTz, "2018-09-22T12:00:00-04:00")
	correctTime := base.Rfc3339Timestamp(correctTimestamp)
	assert.NoError(t, err)

	host := Host{
		StaleTimestamp: &correctTime,
		Reporter:       "puptoo"}
	return &host
}

func TestUpdateSystemPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)
	accountID2 := getOrCreateTestAccount(t)
	modulesList := []vmaas.UpdatesV3RequestModulesList{}
	req := vmaas.UpdatesV3Request{
		PackageList:    []string{"package0"},
		RepositoryList: utils.PtrSliceString([]string{"repo1", "repo2", "repo3"}),
		ModulesList:    &modulesList,
		Releasever:     utils.PtrString("7Server"),
		Basearch:       utils.PtrString("x86_64"),
	}

	sys1, err := updateSystemPlatform(database.Db, id, accountID1, createTestInvHost(t), &req)
	assert.Nil(t, err)

	reporterID1 := 1
	assertSystemInDB(t, id, &accountID1, &reporterID1)
	assertReposInDB(t, req.GetRepositoryList())

	host2 := createTestInvHost(t)
	host2.Reporter = "yupana"
	req.PackageList = []string{"package0", "package1"}
	sys2, err := updateSystemPlatform(database.Db, id, accountID2, host2, &req)
	assert.Nil(t, err)

	reporterID2 := 3
	assertSystemInDB(t, id, &accountID2, &reporterID2)

	assert.Equal(t, sys1.ID, sys2.ID)
	assert.Equal(t, sys1.InventoryID, sys2.InventoryID)
	assert.Equal(t, sys1.Stale, sys2.Stale)
	assert.NotNil(t, sys1.StaleTimestamp)
	assert.Nil(t, sys1.StaleWarningTimestamp)
	assert.Nil(t, sys1.CulledTimestamp)

	deleteData(t)
}

func TestUploadHandler(t *testing.T) {
	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	deleteData(t)

	_ = getOrCreateTestAccount(t)
	event := createTestUploadEvent(id, id, "puptoo", true)
	err := HandleUpload(event)
	assert.NoError(t, err)

	reporterID := 1
	assertSystemInDB(t, id, nil, &reporterID)

	var sys models.SystemPlatform

	assert.NoError(t, database.Db.Where("inventory_id::text = ?", id).Find(&sys).Error)
	after := time.Now().Add(time.Hour)
	sys.LastEvaluation = &after
	assert.NoError(t, database.Db.Save(&sys).Error)
	// Test that second upload did not cause re-evaluation
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	err = HandleUpload(event)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(logHook.LogEntries))
	assert.Equal(t, UploadSuccessNoEval, logHook.LogEntries[1].Message)
	deleteData(t)
}

func TestUploadHandlerWarn(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", id, "puptoo", false)
	err := HandleUpload(noPkgsEvent)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(logHook.LogEntries))
	assert.Equal(t, WarnSkippingNoPackages, logHook.LogEntries[0].Message)
}

func TestUploadHandlerWarnSkipReporter(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", id, "yupana", false)
	err := HandleUpload(noPkgsEvent)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(logHook.LogEntries))
	assert.Equal(t, WarnSkippingReporter, logHook.LogEntries[0].Message)
}

// error when parsing identity
func TestUploadHandlerError1(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent("1", id, "puptoo", true)
	event.Host.Account = ""
	err := HandleUpload(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(logHook.LogEntries))
	assert.Equal(t, ErrorNoAccountProvided, logHook.LogEntries[0].Message)
}

type erroringWriter struct{}

func (t *erroringWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return errors.New("err")
}

// error when processing upload
func TestUploadHandlerError2(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()
	deleteData(t)
	evalWriter = &erroringWriter{}
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	_ = getOrCreateTestAccount(t)
	event := createTestUploadEvent("1", id, "puptoo", true)
	err := HandleUpload(event)
	assert.Error(t, err)
	assert.Equal(t, ErrorKafkaSend, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
	deleteData(t)
}

func TestEnsureReposInDB(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1", "repo10", "repo20"}
	repoIDs, nAdded, err := ensureReposInDB(database.Db, repos)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), nAdded)
	assert.Equal(t, 3, len(repoIDs))
	assertReposInDB(t, repos)
	deleteData(t)
}

func TestEnsureReposEmpty(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	var repos []string
	repoIDs, nAdded, err := ensureReposInDB(database.Db, repos)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), nAdded)
	assert.Equal(t, 0, len(repoIDs))
}

func TestUpdateSystemRepos1(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	deleteData(t)

	systemID := 5
	rhAccountID := 1
	database.Db.Create(models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: 2})

	repos := []string{"repo1", "repo10", "repo20"}
	repoIDs, nReposAdded, err := ensureReposInDB(database.Db, repos)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(repoIDs))
	assert.Equal(t, int64(2), nReposAdded)

	nAdded, nDeleted, err := updateSystemRepos(database.Db, rhAccountID, systemID, repoIDs)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), nAdded)
	assert.Equal(t, int64(1), nDeleted)
	deleteData(t)
}

func TestUpdateSystemRepos2(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	deleteData(t)

	systemID := 5
	rhAccountID := 1
	database.Db.Create(models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{RhAccountID: rhAccountID, SystemID: systemID, RepoID: 2})

	nAdded, nDeleted, err := updateSystemRepos(database.Db, rhAccountID, systemID, []int{})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), nAdded)
	assert.Equal(t, int64(2), nDeleted)
	deleteData(t)
}
