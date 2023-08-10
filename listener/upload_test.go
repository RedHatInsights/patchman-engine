package listener

import (
	"app/base/api"
	"app/base/core"
	"app/base/database"
	"app/base/inventory"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func assertInLogs(t *testing.T, msg string, logs ...log.Entry) {
	nLogs := len(logs)
	i := 0
	for i = 0; i < nLogs; i++ {
		if logs[i].Message == msg {
			assert.Equal(t, msg, logs[i].Message)
			break
		}
	}
	if i == nLogs {
		assert.Fail(t, fmt.Sprintf("Log not found: %s", msg))
	}
}

func createTestInvHost(t *testing.T) *Host {
	correctTimestamp, err := time.Parse(types.Rfc3339NoTz, "2018-09-22T12:00:00-04:00")
	correctTime := types.Rfc3339Timestamp(correctTimestamp)
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
		RepositoryList: []string{"repo1", "repo2", "repo3"},
		ModulesList:    &modulesList,
		Releasever:     utils.PtrString("7Server"),
		Basearch:       utils.PtrString("x86_64"),
	}

	sys1, err := updateSystemPlatform(database.Db, id, accountID1, createTestInvHost(t), nil, &req)
	assert.Nil(t, err)

	reporterID1 := 1
	assertSystemInDB(t, id, &accountID1, &reporterID1)
	assertReposInDB(t, req.RepositoryList)

	host2 := createTestInvHost(t)
	host2.Reporter = "yupana"
	req.PackageList = []string{"package0", "package1"}
	sys2, err := updateSystemPlatform(database.Db, id, accountID2, host2, nil, &req)
	assert.Nil(t, err)

	reporterID2 := 3
	assertSystemInDB(t, id, &accountID2, &reporterID2)

	assert.Equal(t, sys1.ID, sys2.ID)
	assert.Equal(t, sys1.InventoryID, sys2.InventoryID)
	assert.Equal(t, sys1.Stale, sys2.Stale)
	assert.Equal(t, sys1.SatelliteManaged, sys2.SatelliteManaged)
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
	event := createTestUploadEvent(id, id, id, "puptoo", true, false)

	event.Host.SystemProfile.OperatingSystem = inventory.OperatingSystem{Major: 8}
	repos := append(event.Host.SystemProfile.GetYumRepos(), inventory.YumRepo{ID: "epel", Enabled: true})
	event.Host.SystemProfile.YumRepos = &repos

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
	assertInLogs(t, UploadSuccessNoEval, logHook.LogEntries...)
	assertSystemReposInDB(t, sys.ID, []string{"epel-8"})
	deleteData(t)
}

func TestUploadHandlerWarn(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", "1", id, "puptoo", false, false)
	err := HandleUpload(noPkgsEvent)
	assert.NoError(t, err)
	assertInLogs(t, WarnSkippingNoPackages, logHook.LogEntries...)
}

func TestUploadHandlerWarnSkipReporter(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", "1", id, "yupana", false, false)
	err := HandleUpload(noPkgsEvent)
	assert.NoError(t, err)
	assertInLogs(t, WarnSkippingReporter, logHook.LogEntries...)
}

func TestUploadHandlerWarnSkipHostType(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent("1", "1", id, "puptoo", true, false)
	event.Host.SystemProfile.HostType = "edge"
	err := HandleUpload(event)
	assert.NoError(t, err)
	assertInLogs(t, WarnSkippingHostType, logHook.LogEntries...)
}

// error when parsing identity
func TestUploadHandlerError1(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent("1", "1", id, "puptoo", true, false)
	*event.Host.OrgID = ""
	err := HandleUpload(event)
	assert.NoError(t, err)
	assertInLogs(t, ErrorNoAccountProvided, logHook.LogEntries...)
}

type erroringWriter struct{}

func (t *erroringWriter) WriteMessages(_ context.Context, _ ...mqueue.KafkaMessage) error {
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
	event := createTestUploadEvent("1", "1", id, "puptoo", true, false)
	err := HandleUpload(event)
	assert.Nil(t, err)
	time.Sleep(2 * uploadEvalTimeout)
	assertInLogs(t, ErrorKafkaSend, logHook.LogEntries...)
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

	systemID := int64(5)
	rhAccountID := 1
	database.Db.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 2})

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

	systemID := int64(5)
	rhAccountID := 1
	database.Db.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 2})

	nAdded, nDeleted, err := updateSystemRepos(database.Db, rhAccountID, systemID, []int64{})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), nAdded)
	assert.Equal(t, int64(2), nDeleted)
	deleteData(t)
}

func TestFixEpelRepos(t *testing.T) {
	repos := []string{"epel"}
	var sys = inventory.SystemProfile{}
	repos = fixEpelRepos(&sys, repos)
	assert.Equal(t, "epel", repos[0])
}

func TestUpdateSystemPlatformYumUpdates(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)

	httpClient = &api.Client{
		HTTPClient: &http.Client{},
		Debug:      true,
	}
	hostEvent := createTestUploadEvent("1", "1", id, "puptoo", false, true)
	yumUpdates, err := getYumUpdates(hostEvent, httpClient)
	assert.Nil(t, err)

	req := vmaas.UpdatesV3Request{}

	_, err = updateSystemPlatform(database.Db, id, accountID1, createTestInvHost(t), yumUpdates, &req)
	assert.Nil(t, err)

	reporterID1 := 1
	assertSystemInDB(t, id, &accountID1, &reporterID1)
	assertReposInDB(t, req.RepositoryList)
	assertYumUpdatesInDB(t, id, yumUpdates)

	// check that yumUpdates has been updated
	yumUpdates = []byte("{}")
	_, err = updateSystemPlatform(database.Db, id, accountID1, createTestInvHost(t), yumUpdates, &req)
	assert.Nil(t, err)
	assertYumUpdatesInDB(t, id, yumUpdates)

	deleteData(t)
}

// nolint: funlen
func TestStoreOrUpdateSysPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	var oldCount, newCount int
	var nextval, currval int
	database.Db.Model(&models.SystemPlatform{}).Select("count(*)").Find(&oldCount)
	database.Db.Raw("select nextval('system_platform_id_seq')").Find(&nextval)

	colsToUpdate := []string{"vmaas_json", "json_checksum", "reporter_id", "satellite_managed"}
	json := "this_is_json"
	inStore := models.SystemPlatform{
		InventoryID:      "99990000-0000-0000-0000-000000000001",
		RhAccountID:      1,
		VmaasJSON:        &json,
		DisplayName:      "display_name",
		SatelliteManaged: false,
	}
	// insert new row
	err := storeOrUpdateSysPlatform(database.Db, &inStore, colsToUpdate)
	assert.Nil(t, err)

	var outStore models.SystemPlatform
	database.Db.Model(models.SystemPlatform{}).Find(&outStore, inStore.ID)
	defer database.Db.Model(models.SystemPlatform{}).Delete(outStore)

	assert.Equal(t, inStore.InventoryID, outStore.InventoryID)
	assert.Equal(t, inStore.RhAccountID, outStore.RhAccountID)
	assert.Equal(t, *inStore.VmaasJSON, *outStore.VmaasJSON)
	assert.Equal(t, inStore.SatelliteManaged, outStore.SatelliteManaged)

	updateJSON := "updated_json"
	reporter := 2
	inUpdate := outStore
	inUpdate.VmaasJSON = &updateJSON
	inUpdate.JSONChecksum = &updateJSON
	inUpdate.ReporterID = &reporter
	inUpdate.DisplayName = "should_not_be_updated"
	inUpdate.SatelliteManaged = true

	// update row
	err = storeOrUpdateSysPlatform(database.Db, &inUpdate, colsToUpdate)
	assert.Nil(t, err)

	var outUpdate models.SystemPlatform
	database.Db.Model(models.SystemPlatform{}).Find(&outUpdate, inUpdate.ID)
	assert.Equal(t, inUpdate.InventoryID, outUpdate.InventoryID)
	assert.Equal(t, inUpdate.RhAccountID, outUpdate.RhAccountID)
	assert.Equal(t, *inUpdate.VmaasJSON, *outUpdate.VmaasJSON)
	assert.Equal(t, *inUpdate.JSONChecksum, *outUpdate.JSONChecksum)
	assert.Equal(t, *inUpdate.ReporterID, *outUpdate.ReporterID)
	assert.Equal(t, inUpdate.SatelliteManaged, outUpdate.SatelliteManaged)
	// it should update the row
	assert.Equal(t, outStore.ID, outUpdate.ID)
	// DisplayName is not in colsToUpdate, it should not be updated
	assert.Equal(t, outStore.DisplayName, outUpdate.DisplayName)

	// make sure we are not creating gaps in id sequences
	database.Db.Model(&models.SystemPlatform{}).Select("count(*)").Find(&newCount)
	database.Db.Raw("select currval('system_platform_id_seq')").Find(&currval)
	countInc := newCount - oldCount
	maxInc := currval - nextval
	assert.Equal(t, countInc, maxInc)
}
