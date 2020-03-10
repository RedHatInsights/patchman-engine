package listener

import (
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

func createTestInvHost() *Host {
	correctTimestamp := "2018-09-22T12:00:00-04:00"
	wrongTimestamp := "x018-09-22T12:00:00-04:00"

	host := Host{StaleTimestamp: &correctTimestamp, StaleWarningTimestamp: &wrongTimestamp}
	return &host
}

func TestUpdateSystemPlatform(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	deleteData(t)

	accountID1 := getOrCreateTestAccount(t)
	accountID2 := getOrCreateTestAccount(t)
	req := vmaas.UpdatesV3Request{
		PackageList:    []string{"package0"},
		RepositoryList: []string{},
		ModulesList:    []vmaas.UpdatesRequestModulesList{},
		Releasever:     "7Server",
		Basearch:       "x86_64",
	}

	sys1, err := updateSystemPlatform(id, accountID1, createTestInvHost(), &req)
	assert.Nil(t, err)

	assertSystemInDb(t, id, &accountID1)

	sys2, err := updateSystemPlatform(id, accountID2, createTestInvHost(), &req)
	assert.Nil(t, err)

	assertSystemInDb(t, id, &accountID2)

	assert.Equal(t, sys1.ID, sys2.ID)
	assert.Equal(t, sys1.InventoryID, sys2.InventoryID)
	assert.Equal(t, sys1.JSONChecksum, sys2.JSONChecksum)
	assert.Equal(t, sys1.OptOut, sys2.OptOut)
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
	event := createTestUploadEvent(id, true)
	uploadHandler(event)

	assertSystemInDb(t, id, nil)
	deleteData(t)
}

// error when parsing identity
func TestUploadHandlerWarn(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	noPkgsEvent := createTestUploadEvent(id, false)
	uploadHandler(noPkgsEvent)
	assert.Equal(t, 1, len(logHook.LogEntries))
	assert.Equal(t, WarnSkippingNoPackages, logHook.LogEntries[0].Message)
}

// error when parsing identity
func TestUploadHandlerError1(t *testing.T) {
	logHook := utils.NewTestLogHook()
	log.AddHook(logHook)
	event := createTestUploadEvent(id, true)
	event.Host.Account = ""
	uploadHandler(event)
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
	event := createTestUploadEvent(id, true)
	uploadHandler(event)
	assert.Equal(t, ErrorProcessUpload, logHook.LogEntries[len(logHook.LogEntries)-1].Message)
	deleteData(t)
}

func TestEnsureReposInDb(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	repos := []string{"repo1", "repo10", "repo20"}
	repoIDs, nAdded, err := ensureReposInDb(database.Db, repos)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), nAdded)
	assert.Equal(t, 3, len(repoIDs))
	assertReposInDb(t, repos)
	deleteData(t)
}

func TestUpdateSystemRepos1(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	deleteData(t)

	systemID := 5
	database.Db.Create(models.SystemRepo{SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{SystemID: systemID, RepoID: 2})

	repos := []string{"repo1", "repo10", "repo20"}
	repoIDs, nReposAdded, err := ensureReposInDb(database.Db, repos)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(repoIDs))
	assert.Equal(t, int64(2), nReposAdded)

	nAdded, nDeleted, err := updateSystemRepos(database.Db, systemID, repoIDs)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), nAdded)
	assert.Equal(t, 1, nDeleted)
	deleteData(t)
}

func TestUpdateSystemRepos2(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	deleteData(t)

	systemID := 5
	database.Db.Create(models.SystemRepo{SystemID: systemID, RepoID: 1})
	database.Db.Create(models.SystemRepo{SystemID: systemID, RepoID: 2})

	nAdded, nDeleted, err := updateSystemRepos(database.Db, systemID, []int{})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), nAdded)
	assert.Equal(t, 2, nDeleted)
	deleteData(t)
}
