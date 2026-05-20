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

	"github.com/google/uuid"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var accountID = int(1)

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

	now := time.Now()
	host := Host{
		ID:             id,
		StaleTimestamp: &correctTime,
		Reporter:       "puptoo",
		PerReporterStaleness: map[string]inventory.ReporterStaleness{
			"puptoo": {LastCheckIn: types.Rfc3339TimestampWithZ(now)},
		},
	}
	return &host
}

func createTestHostWithEnv(reporter, consumer, baseURL string) *Host {
	consumerUUID, err := uuid.Parse(consumer)
	if err != nil {
		consumerUUID = uuid.Nil
	}
	return &Host{
		ID:       id,
		Reporter: reporter,
		SystemProfile: inventory.SystemProfile{
			OwnerID: &consumerUUID,
			YumRepos: &[]inventory.YumRepo{{
				ID:      "base",
				Enabled: true,
				BaseURL: baseURL}},
		},
	}
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

	sys1, err := updateSystemPlatform(database.DB, accountID1, createTestInvHost(t), nil, &req)
	assert.Nil(t, err)

	reporterID1 := 1
	assertSystemInDB(t, id, &accountID1, &reporterID1)
	assertReposInDB(t, req.RepositoryList)

	host2 := createTestInvHost(t)
	host2.Reporter = "yupana"
	req.PackageList = []string{"package0", "package1"}
	sys2, err := updateSystemPlatform(database.DB, accountID2, host2, nil, &req)
	assert.Nil(t, err)

	reporterID2 := 3
	assertSystemInDB(t, id, &accountID2, &reporterID2)

	assert.Equal(t, sys1.Inventory.ID, sys2.Inventory.ID)
	assert.Equal(t, sys1.Inventory.InventoryID, sys2.Inventory.InventoryID)
	assert.Equal(t, sys1.Inventory.Stale, sys2.Inventory.Stale)
	assert.Equal(t, sys1.Inventory.SatelliteManaged, sys2.Inventory.SatelliteManaged)
	assert.NotNil(t, sys1.Inventory.StaleTimestamp)
	assert.Nil(t, sys1.Inventory.StaleWarningTimestamp)
	assert.Nil(t, sys1.Inventory.CulledTimestamp)

	deleteData(t)
}

func TestUpdateSystemPlatformUpdatesSubscriptionManagerIDWhenOwnerIDChanges(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	acc := getOrCreateTestAccount(t)
	correctTimestamp, err := time.Parse(types.Rfc3339NoTz, "2018-09-22T12:00:00-04:00")
	require.NoError(t, err)
	correctTime := types.Rfc3339Timestamp(correctTimestamp)

	modulesSlice := []vmaas.UpdatesV3RequestModulesList{}
	reqProfile := vmaas.UpdatesV3Request{
		PackageList:    []string{"kernel-0:54321-1.rhel8.x86_64"},
		RepositoryList: []string{"repo1", "repo2", "repo3"},
		ModulesList:    &modulesSlice,
		Releasever:     utils.PtrString("7Server"),
		Basearch:       utils.PtrString("x86_64"),
	}

	ev := createTestUploadEvent(id, id, "puptoo", true, false, "created")
	ev.Host.StaleTimestamp = &correctTime
	ownerID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ev.Host.SystemProfile.OwnerID = &ownerID1

	_, err = updateSystemPlatform(database.DB, acc, &ev.Host, nil, &reqProfile)
	require.NoError(t, err)
	assertSystemInventoryProfileMatchesHost(t, id, &ev.Host)

	ownerID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ev.Host.SystemProfile.OwnerID = &ownerID2

	_, err = updateSystemPlatform(database.DB, acc, &ev.Host, nil, &reqProfile)
	require.NoError(t, err)
	assertSystemInventoryProfileMatchesHost(t, id, &ev.Host)

	ev.Host.SystemProfile.OwnerID = nil
	_, err = updateSystemPlatform(database.DB, acc, &ev.Host, nil, &reqProfile)
	require.NoError(t, err)
	assertSystemInventoryProfileMatchesHost(t, id, &ev.Host)

	deleteData(t)
}

func TestUpdateSystemPlatformRefreshesInventoryProfileOnConflict(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	deleteData(t)
	acc := getOrCreateTestAccount(t)
	correctTimestamp, err := time.Parse(types.Rfc3339NoTz, "2018-09-22T12:00:00-04:00")
	require.NoError(t, err)
	correctTime := types.Rfc3339Timestamp(correctTimestamp)
	modulesSlice := []vmaas.UpdatesV3RequestModulesList{}
	reqProfile := vmaas.UpdatesV3Request{
		PackageList:    []string{"kernel-0:54321-1.rhel8.x86_64"},
		RepositoryList: []string{"repo1", "repo2", "repo3"},
		ModulesList:    &modulesSlice,
		Releasever:     utils.PtrString("7Server"),
		Basearch:       utils.PtrString("x86_64"),
	}
	ev := createTestUploadEvent(id, id, "puptoo", true, false, "created")
	ev.Host.StaleTimestamp = &correctTime
	_, err = updateSystemPlatform(database.DB, acc, &ev.Host, nil, &reqProfile)
	require.NoError(t, err)
	assertSystemInventoryProfileMatchesHost(t, id, &ev.Host)

	ev.Host.Tags = []byte(`{"namespace": "insights-client","key": "env","value": "staging"}`)
	ev.Host.SystemProfile.OperatingSystem.Minor = 5
	ev.Host.SystemProfile.Rhsm.Version = "8.5"
	ev.Host.SystemProfile.Workloads.Sap.Sids = []string{"sid9"}
	ev.Host.SystemProfile.Workloads.Ansible.ControllerVersion = "2.13.0"
	ev.Host.SystemProfile.Workloads.Mssql.Version = "16.0"

	_, err = updateSystemPlatform(database.DB, acc, &ev.Host, nil, &reqProfile)
	require.NoError(t, err)
	assertSystemInventoryProfileMatchesHost(t, id, &ev.Host)

	deleteData(t)
}

func TestUploadHandlerCreatedSystem(t *testing.T) {
	eventTypes := []string{"created", "updated"}
	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			utils.SkipWithoutDB(t)
			utils.SkipWithoutPlatform(t)
			core.SetupTestEnvironment()
			configure()
			deleteData(t)

			_ = getOrCreateTestAccount(t)
			event := createTestUploadEvent(id, id, "puptoo", true, false, eventType)

			event.Host.SystemProfile.OperatingSystem = inventory.OperatingSystem{Major: 8}
			repos := append(event.Host.SystemProfile.GetYumRepos(), inventory.YumRepo{ID: "epel", Enabled: true})
			event.Host.SystemProfile.YumRepos = &repos

			err := HandleUpload(event)
			assert.NoError(t, err)

			reporterID := 1
			assertSystemInDB(t, id, nil, &reporterID)

			var inv models.SystemInventory
			assert.NoError(t, database.DB.Where("inventory_id = ?::uuid", id).First(&inv).Error)
			var patch models.SystemPatch
			assert.NoError(t, database.DB.Where("rh_account_id = ? AND system_id = ?", inv.RhAccountID, inv.ID).
				First(&patch).Error)
			after := time.Now().Add(time.Hour)
			patch.LastEvaluation = &after
			assert.NoError(t, database.DB.Save(&patch).Error)
			// Test that second upload did not cause re-evaluation
			logHook := utils.NewTestLogHook()
			log.AddHook(logHook)
			err = HandleUpload(event)
			assert.NoError(t, err)
			assertInLogs(t, UploadSuccessNoEval, logHook.LogEntries...)
			assertSystemReposInDB(t, inv.ID, []string{"epel-8"})
			deleteData(t)
		})
	}
}

func TestUploadHandlerWarn(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", id, "puptoo", false, false, "created")
	err := HandleUpload(noPkgsEvent)
	if assert.Error(t, err) {
		assert.ErrorIs(t, err, ErrNoPackages)
	}
	assertInLogs(t, ErrNoPackages.Error(), logHook.LogEntries...)
}

func TestUploadHandlerWarnSkipReporter(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent("1", id, "yupana", false, false, "created")
	err := HandleUpload(noPkgsEvent)
	if assert.Error(t, err) {
		assert.ErrorIs(t, err, ErrReporter)
	}
	assertInLogs(t, ErrReporter.Error(), logHook.LogEntries...)
}

func TestUploadHandlerWarnSkipHostType(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent("1", id, "puptoo", true, false, "created")
	event.Host.SystemProfile.HostType = "edge"
	err := HandleUpload(event)
	if assert.Error(t, err) {
		assert.ErrorIs(t, err, ErrHostType)
	}
	assertInLogs(t, ErrHostType.Error(), logHook.LogEntries...)
}

// error when parsing identity
func TestUploadHandlerError1(t *testing.T) {
	utils.SkipWithoutDB(t)
	configure()
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent("1", id, "puptoo", true, false, "created")
	*event.Host.OrgID = ""
	err := HandleUpload(event)
	if assert.Error(t, err) {
		assert.ErrorIs(t, err, ErrNoAccountProvided)
	}
	assertInLogs(t, ErrNoAccountProvided.Error(), logHook.LogEntries...)
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
	createdSystemsWriter = &erroringWriter{}
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	_ = getOrCreateTestAccount(t)
	event := createTestUploadEvent("1", id, "puptoo", true, false, "created")
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
	repoIDs, nAdded, err := ensureReposInDB(database.DB, repos)
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
	repoIDs, nAdded, err := ensureReposInDB(database.DB, repos)
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
	database.DB.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 1})
	database.DB.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 2})

	repos := []string{"repo1", "repo10", "repo20"}
	repoIDs, nReposAdded, err := ensureReposInDB(database.DB, repos)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(repoIDs))
	assert.Equal(t, int64(2), nReposAdded)

	nAdded, nDeleted, err := updateSystemRepos(database.DB, rhAccountID, systemID, repoIDs)
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
	database.DB.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 1})
	database.DB.Create(models.SystemRepo{RhAccountID: int64(rhAccountID), SystemID: systemID, RepoID: 2})

	nAdded, nDeleted, err := updateSystemRepos(database.DB, rhAccountID, systemID, []int64{})
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

	reposContent := []string{"EPEL_9_Everything_x86_64"}
	var sysContent = inventory.SystemProfile{OperatingSystem: inventory.OperatingSystem{Major: 9}}
	reposContent = fixEpelRepos(&sysContent, reposContent)
	assert.Equal(t, "epel-9", reposContent[0])
}

// nolint: funlen
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
	hostEvent := createTestUploadEvent("1", id, "puptoo", false, true, "created")
	yumUpdates, err := getYumUpdates(hostEvent, httpClient)
	assert.Nil(t, err)

	req := vmaas.UpdatesV3Request{}

	_, err = updateSystemPlatform(database.DB, accountID1, &hostEvent.Host, yumUpdates, &req)
	assert.Nil(t, err)

	reporterID1 := 1
	assertSystemInDB(t, id, &accountID1, &reporterID1)
	assertReposInDB(t, req.RepositoryList)
	assertYumUpdatesInDB(t, id, yumUpdates)
	assertSystemInventoryProfileMatchesHost(t, id, &hostEvent.Host)

	// check that yumUpdates has been updated (keep the same Host so profile columns are not wiped)
	yumUpdates.RawParsed = []byte("{}")
	_, err = updateSystemPlatform(database.DB, accountID1, &hostEvent.Host, yumUpdates, &req)
	assert.Nil(t, err)
	assertYumUpdatesInDB(t, id, yumUpdates)
	assertSystemInventoryProfileMatchesHost(t, id, &hostEvent.Host)

	hostEvent.Host.Tags = []byte(`{"namespace": "insights-client","key": "env","value": "staging"}`)
	hostEvent.Host.SystemProfile.OperatingSystem.Minor = 6
	hostEvent.Host.SystemProfile.Rhsm.Version = "8.6"
	hostEvent.Host.SystemProfile.Workloads.Sap.Sids = []string{"sid-yum-test"}
	hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion = "2.14.0"
	hostEvent.Host.SystemProfile.Workloads.Mssql.Version = "17.0"
	_, err = updateSystemPlatform(database.DB, accountID1, &hostEvent.Host, yumUpdates, &req)
	assert.Nil(t, err)
	assertYumUpdatesInDB(t, id, yumUpdates)
	assertSystemInventoryProfileMatchesHost(t, id, &hostEvent.Host)

	// Clear workload-related profile fields and verify ON CONFLICT updates clear them in DB.
	hostEvent.Host.SystemProfile.Rhsm.Version = ""
	hostEvent.Host.SystemProfile.Workloads.Sap.SapSystem = false
	hostEvent.Host.SystemProfile.Workloads.Sap.Sids = nil
	hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion = ""
	hostEvent.Host.SystemProfile.Workloads.Mssql.Version = ""
	_, err = updateSystemPlatform(database.DB, accountID1, &hostEvent.Host, yumUpdates, &req)
	assert.Nil(t, err)
	assertYumUpdatesInDB(t, id, yumUpdates)
	assertSystemInventoryProfileMatchesHost(t, id, &hostEvent.Host)

	deleteData(t)
}

// nolint: funlen
func TestStoreOrUpdateSysPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	var oldCount, newCount int
	var nextval, currval int
	database.DB.Model(&models.SystemInventory{}).Select("count(*)").Find(&oldCount)
	database.DB.Raw("select nextval('system_inventory_id_seq')").Find(&nextval)

	colsToUpdate := []string{"vmaas_json", "json_checksum", "reporter_id", "satellite_managed"}
	vmaasJSON := "this_is_json"
	// insert new row
	hostEvent := createTestUploadEvent("1", id, "puptoo", false, true, "created")
	hostWorkspaces := inventory.Groups(hostEvent.Host.Groups)
	inStore := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			InventoryID:                      "99990000-0000-0000-0000-000000000001",
			RhAccountID:                      1,
			VmaasJSON:                        &vmaasJSON,
			DisplayName:                      "display_name",
			SatelliteManaged:                 false,
			Created:                          hostEvent.Host.Created,
			Tags:                             utils.MarshalNilToJSONB(hostEvent.Host.Tags),
			Workspaces:                       &hostWorkspaces,
			OSName:                           utils.EmptyToNil(&hostEvent.Host.SystemProfile.OperatingSystem.Name),
			OSMajor:                          &hostEvent.Host.SystemProfile.OperatingSystem.Major,
			OSMinor:                          &hostEvent.Host.SystemProfile.OperatingSystem.Minor,
			RhsmVersion:                      utils.EmptyToNil(&hostEvent.Host.SystemProfile.Rhsm.Version),
			SubscriptionManagerID:            hostEvent.Host.SystemProfile.OwnerID,
			SapWorkload:                      hostEvent.Host.SystemProfile.Workloads.Sap.SapSystem,
			SapWorkloadSIDs:                  pq.StringArray(hostEvent.Host.SystemProfile.Workloads.Sap.Sids),
			AnsibleWorkload:                  hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion != "",
			AnsibleWorkloadControllerVersion: utils.EmptyToNil(&hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion), // nolint:lll
			MssqlWorkload:                    hostEvent.Host.SystemProfile.Workloads.Mssql.Version != "",
			MssqlWorkloadVersion:             utils.EmptyToNil(&hostEvent.Host.SystemProfile.Workloads.Mssql.Version),
		},
		Patch: models.SystemPatch{RhAccountID: 1},
	}

	err := storeOrUpdateSysPlatform(database.DB, inStore, colsToUpdate)
	assert.Nil(t, err)

	var outStore models.SystemInventory
	assert.NoError(t, database.DB.Where("id = ? AND rh_account_id = ?", inStore.Inventory.ID, inStore.Inventory.RhAccountID). // nolint:lll
																	First(&outStore).Error)
	defer func() {
		database.DB.Unscoped().Where("rh_account_id = ? AND system_id = ?", outStore.RhAccountID, outStore.ID).
			Delete(&models.SystemPatch{})
		database.DB.Unscoped().Where("id = ? AND rh_account_id = ?", outStore.ID, outStore.RhAccountID).
			Delete(&models.SystemInventory{})
	}()

	assert.Equal(t, inStore.Inventory.InventoryID, outStore.InventoryID)
	assert.Equal(t, inStore.Inventory.RhAccountID, outStore.RhAccountID)
	assert.Equal(t, *inStore.Inventory.VmaasJSON, *outStore.VmaasJSON)
	assert.Equal(t, inStore.Inventory.SatelliteManaged, outStore.SatelliteManaged)

	// verify SystemInventory was created from Host fields
	var inventoryAfterInsert models.SystemInventory
	err = database.DB.Where("id = ?", inStore.Inventory.ID).First(&inventoryAfterInsert).Error
	assert.Nil(t, err)

	assert.Contains(t, string(inventoryAfterInsert.Tags), `"namespace": "insights-client"`)
	assert.Contains(t, string(inventoryAfterInsert.Tags), `"key": "env"`)
	assert.Contains(t, string(inventoryAfterInsert.Tags), `"value": "prod"`)

	require.NotNil(t, inventoryAfterInsert.Workspaces)
	assert.Equal(t, hostEvent.Host.Groups, []inventory.Group(*inventoryAfterInsert.Workspaces))

	assert.Equal(t, hostEvent.Host.SystemProfile.OperatingSystem.Name, *inventoryAfterInsert.OSName)
	assert.Equal(t, hostEvent.Host.SystemProfile.OperatingSystem.Major, *inventoryAfterInsert.OSMajor)
	assert.Equal(t, hostEvent.Host.SystemProfile.OperatingSystem.Minor, *inventoryAfterInsert.OSMinor)

	assert.Equal(t, hostEvent.Host.SystemProfile.Rhsm.Version, *inventoryAfterInsert.RhsmVersion)

	assert.Equal(t, hostEvent.Host.SystemProfile.Workloads.Sap.SapSystem, inventoryAfterInsert.SapWorkload)
	assert.ElementsMatch(t,
		pq.StringArray(hostEvent.Host.SystemProfile.Workloads.Sap.Sids),
		inventoryAfterInsert.SapWorkloadSIDs)

	assert.Equal(t, true, inventoryAfterInsert.AnsibleWorkload)
	assert.Equal(t,
		hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion,
		*inventoryAfterInsert.AnsibleWorkloadControllerVersion)

	assert.Equal(t, true, inventoryAfterInsert.MssqlWorkload)
	assert.Equal(t, hostEvent.Host.SystemProfile.Workloads.Mssql.Version, *inventoryAfterInsert.MssqlWorkloadVersion)

	updateJSON := "updated_json"
	reporter := 2
	var patchAfterInsert models.SystemPatch
	assert.NoError(t, database.DB.Where("rh_account_id = ? AND system_id = ?", outStore.RhAccountID, outStore.ID).
		First(&patchAfterInsert).Error)
	inUpdate := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			ID:                               outStore.ID,
			InventoryID:                      outStore.InventoryID,
			RhAccountID:                      outStore.RhAccountID,
			VmaasJSON:                        &updateJSON,
			JSONChecksum:                     &updateJSON,
			ReporterID:                       &reporter,
			DisplayName:                      "should_not_be_updated",
			SatelliteManaged:                 true,
			Created:                          hostEvent.Host.Created,
			Tags:                             utils.MarshalNilToJSONB(hostEvent.Host.Tags),
			Workspaces:                       &hostWorkspaces,
			OSName:                           utils.EmptyToNil(&hostEvent.Host.SystemProfile.OperatingSystem.Name),
			OSMajor:                          &hostEvent.Host.SystemProfile.OperatingSystem.Major,
			OSMinor:                          &hostEvent.Host.SystemProfile.OperatingSystem.Minor,
			RhsmVersion:                      utils.EmptyToNil(&hostEvent.Host.SystemProfile.Rhsm.Version),
			SubscriptionManagerID:            hostEvent.Host.SystemProfile.OwnerID,
			SapWorkload:                      hostEvent.Host.SystemProfile.Workloads.Sap.SapSystem,
			SapWorkloadSIDs:                  pq.StringArray(hostEvent.Host.SystemProfile.Workloads.Sap.Sids),
			AnsibleWorkload:                  hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion != "",
			AnsibleWorkloadControllerVersion: utils.EmptyToNil(&hostEvent.Host.SystemProfile.Workloads.Ansible.ControllerVersion), // nolint:lll
			MssqlWorkload:                    hostEvent.Host.SystemProfile.Workloads.Mssql.Version != "",
			MssqlWorkloadVersion:             utils.EmptyToNil(&hostEvent.Host.SystemProfile.Workloads.Mssql.Version),
		},
		Patch: models.SystemPatch{
			RhAccountID: outStore.RhAccountID,
			TemplateID:  patchAfterInsert.TemplateID,
		},
	}

	// update row
	err = storeOrUpdateSysPlatform(database.DB, inUpdate, colsToUpdate)
	require.NoError(t, err)

	var outUpdate models.SystemInventory
	assert.NoError(t, database.DB.Where("id = ? AND rh_account_id = ?", inUpdate.Inventory.ID, inUpdate.Inventory.RhAccountID). // nolint:lll
																	First(&outUpdate).Error)
	assert.Equal(t, inUpdate.Inventory.InventoryID, outUpdate.InventoryID)
	assert.Equal(t, inUpdate.Inventory.RhAccountID, outUpdate.RhAccountID)
	require.NotNil(t, outUpdate.VmaasJSON)
	assert.Equal(t, *inUpdate.Inventory.VmaasJSON, *outUpdate.VmaasJSON)
	require.NotNil(t, outUpdate.JSONChecksum)
	assert.Equal(t, *inUpdate.Inventory.JSONChecksum, *outUpdate.JSONChecksum)
	require.NotNil(t, outUpdate.ReporterID)
	assert.Equal(t, *inUpdate.Inventory.ReporterID, *outUpdate.ReporterID)
	assert.Equal(t, inUpdate.Inventory.SatelliteManaged, outUpdate.SatelliteManaged)
	// it should update the row
	assert.Equal(t, outStore.ID, outUpdate.ID)
	// DisplayName is not in colsToUpdate, it should not be updated
	assert.Equal(t, outStore.DisplayName, outUpdate.DisplayName)

	// make sure we are not creating gaps in id sequences
	database.DB.Model(&models.SystemInventory{}).Select("count(*)").Find(&newCount)
	database.DB.Raw("select currval('system_inventory_id_seq')").Find(&currval)
	countInc := newCount - oldCount
	maxInc := currval - nextval
	assert.Equal(t, countInc, maxInc)
}

func TestGetRepoPath(t *testing.T) {
	repoPath, err := getRepoPath(nil, nil)
	assert.Nil(t, err)
	assert.Empty(t, repoPath)

	repo := inventory.YumRepo{}
	sp := inventory.SystemProfile{}

	repoPath, err = getRepoPath(&sp, &repo)
	assert.Nil(t, err)
	assert.Empty(t, repoPath)

	repo = inventory.YumRepo{Mirrorlist: "://asdf"}
	repoPath, err = getRepoPath(&sp, &repo)
	assert.NotNil(t, err)
	assert.Empty(t, repoPath)

	repo = inventory.YumRepo{Mirrorlist: "https://rhui.redhat.com/"}
	repoPath, err = getRepoPath(&sp, &repo)
	assert.Nil(t, err)
	assert.Empty(t, repoPath)

	repo = inventory.YumRepo{BaseURL: "https://rhui.redhat.com/"}
	repoPath, err = getRepoPath(&sp, &repo)
	assert.Nil(t, err)
	assert.Empty(t, repoPath)

	repo = inventory.YumRepo{
		BaseURL: "https://rhui.redhat.com/pulp/mirror/content/dist/rhel8/rhui/8.4/x86_64/baseos/os",
	}
	repoPath, err = getRepoPath(&sp, &repo)
	assert.Nil(t, err)
	assert.Equal(t, "/content/dist/rhel8/rhui/8.4/x86_64/baseos/os", repoPath)

	repo = inventory.YumRepo{
		BaseURL: "https://rhui.redhat.com/pulp/mirror/content/dist/rhel8/rhui/$releasever/$basearch/baseos/os",
	}
	arch := "x86_64"
	release := "8.4"
	sp = inventory.SystemProfile{Arch: &arch, Releasever: &release}
	repoPath, err = getRepoPath(&sp, &repo)
	assert.Nil(t, err)
	assert.Equal(t, "/content/dist/rhel8/rhui/8.4/x86_64/baseos/os", repoPath)
}

func TestHostTemplateRhsmReporter(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	host := createTestHostWithEnv(rhsmReporter, "00000000-0000-0000-0000-000000000001",
		"https://cert.console.example.com/api/pulp-content/abcdef/templates/"+
			"12345678-90ab-cdef-1234-567890abcdef/content/dist/rhel9/$releasever/x86_64/baseos/os")
	templateID := hostTemplate(database.DB, accountID, host)
	assert.NotNil(t, templateID)
	assert.Equal(t, int64(1), *templateID)
}

func TestHostTemplatePuptoo(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	host := createTestHostWithEnv(puptooReporter, "00000000-0000-0000-0000-000000000002",
		"https://cert.console.example.com/api/pulp-content/abcdef/templates/"+
			"12345678-90ab-cdef-1234-567890abcdef/content/dist/rhel9/$releasever/x86_64/baseos/os")
	templateID := hostTemplate(database.DB, accountID, host)
	assert.NotNil(t, templateID)
	assert.Equal(t, int64(2), *templateID)
}

func TestNoHostTemplate(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	host := createTestHostWithEnv(puptooReporter, "00000000-0000-0000-0000-000000000002",
		"https://cdn.example.com/content/dist/rhel9/$releasever/x86_64/baseos/os")
	templateID := hostTemplate(database.DB, accountID, host)
	assert.Nil(t, templateID)
}

func TestHostTemplateCandlepinFailure(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	host := createTestHostWithEnv(puptooReporter, "return_404",
		"https://cert.console.example.com/api/pulp-content/abcdef/templates/"+
			"12345678-90ab-cdef-1234-567890abcdef/content/dist/rhel9/$releasever/x86_64/baseos/os")
	templateID := hostTemplate(database.DB, accountID, host)
	assert.Nil(t, templateID)
}
