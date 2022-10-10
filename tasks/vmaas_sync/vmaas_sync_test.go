package vmaas_sync //nolint:revive,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/types/vmaas"
	"app/base/utils"
	"app/tasks"
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var msgs []mqueue.KafkaMessage

type mockKafkaWriter struct{}

func (t mockKafkaWriter) WriteMessages(_ context.Context, ev ...mqueue.KafkaMessage) error {
	msgs = append(msgs, ev...)
	return nil
}

func TestSync(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	// ensure all repos to be marked "third_party" = true
	assert.NoError(t, database.Db.Table("repo").Where("name IN (?)", []string{"repo1", "repo2", "repo3"}).
		Update("third_party", true).Error)
	database.CheckThirdPartyRepos(t, []string{"repo1", "repo2", "repo3", "repo4"}, true) // ensure testing set

	evalWriter = &mockKafkaWriter{}

	runSync()

	expected := []string{"RH-100"}
	database.CheckAdvisoriesInDB(t, expected)

	var repos []models.Repo
	assert.NoError(t, database.Db.Model(&repos).Error)

	database.CheckThirdPartyRepos(t, []string{"repo1", "repo2", "repo3"}, false) // sync updated the flag
	database.CheckThirdPartyRepos(t, []string{"repo4"}, true)                    // third party repo4 has correct flag

	// For one account we expect a bulk message
	assert.Equal(t, 1, len(msgs))

	ts, err := database.GetTimestampKVValueStr(LastEvalRepoBased) // check updated timestamp
	assert.Nil(t, err)
	assert.Equal(t, time.Now().Format("2006"), (*ts)[0:4])
	resetLastEvalTimestamp(t)
	database.DeleteNewlyAddedPackages(t)
	database.DeleteNewlyAddedAdvisories(t)
}

func TestHandleContextCancel(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	utils.ConfigureLogging()

	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	tasks.HandleContextCancel(func() {})
	base.CancelContext()
	utils.AssertEqualWait(t, 1, func() (exp, act interface{}) {
		return 1, len(hook.LogEntries)
	})
	assert.Equal(t, "stopping vmaas_sync", hook.LogEntries[0].Message)
}

func TestVMaaSErrataCall(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()

	req := vmaas.ErrataRequest{PageSize: 10, ErrataList: []string{".*"}}
	resp := vmaas.ErrataResponse{}
	ctx := context.Background()
	httpResp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasErratasURL, &req, &resp) // nolint: bodyclose
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, 4, len(resp.ErrataList))
}

func TestVMaaSReposCall(t *testing.T) {
	utils.SkipWithoutPlatform(t)
	core.SetupTestEnvironment()
	configure()

	req := vmaas.ReposRequest{PageSize: 10, RepositoryList: []string{".*"}}
	resp := vmaas.ReposResponse{}
	ctx := context.Background()
	httpResp, err := vmaasClient.Request(&ctx, http.MethodPost, vmaasReposURL, &req, &resp) // nolint: bodyclose
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, 3, len(resp.RepositoryList))
}
