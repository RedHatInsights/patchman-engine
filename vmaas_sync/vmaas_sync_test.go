package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var msgs []kafka.Message

type mockKafkaWriter struct{}

func (t mockKafkaWriter) WriteMessages(_ context.Context, ev ...kafka.Message) error {
	msgs = append(msgs, ev...)
	return nil
}

func TestSync(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	// set all repos as thirdparty
	// this tests syncRepos() function
	var expectedRepos = []models.Repo{
		{ID: 1, Name: "repo1", ThirdParty: false},
		{ID: 2, Name: "repo2", ThirdParty: false},
		{ID: 3, Name: "repo3", ThirdParty: false},
		{ID: 4, Name: "repo4", ThirdParty: true},
	}
	// assert.NoError(t, database.Db.Set("gorm:query_option", "ON CONFLICT DO NOTHING").Create(&expectedRepos).Error)
	assert.NoError(t, database.Db.Create(&expectedRepos).Error)
	assert.NoError(t, database.Db.Table("repo").Update("third_party", true).Error)

	evalWriter = &mockKafkaWriter{}

	err := websocketHandler([]byte("webapps-refreshed"), nil)
	assert.Nil(t, err)

	expected := []string{"RH-100"}
	database.CheckAdvisoriesInDB(t, expected)

	evras := []string{"5.10.13-200.fc31.x86_64"}
	assert.NoError(t, database.Db.Unscoped().Where("evra in (?)", evras).Delete(&models.Package{}).Error)
	assert.NoError(t, database.Db.Unscoped().Where("name IN (?)", expected).Delete(&models.AdvisoryMetadata{}).Error)

	var repos []models.Repo
	assert.NoError(t, database.Db.Model(&repos).Error)
	assert.Equal(t, expectedRepos, repos)
	//	assert.NoError(t, database.Db.Table("repo").Select("id").Where("third_party = false").Scan(&repos).Error)
	//	assert.Equal(t, redhatRepos, repos)
	//	thirdPartyRepos := []int{4}
	//	assert.NoError(t, database.Db.Table("repo").Select("id").Where("third_party = true").Scan(&repos).Error)
	//	assert.Equal(t, thirdPartyRepos, repos)

	// For one account we expect a bulk message
	assert.Equal(t, 1, len(msgs))

	ts, err := getLastRepobasedEvalTms() // check updated timestamp
	assert.Nil(t, err)
	assert.Equal(t, time.Now().Year(), ts.Year())
	resetLastEvalTimestamp(t)
}

func TestHandleContextCancel(t *testing.T) {
	assert.Nil(t, os.Setenv("LOG_STYLE", "json"))
	utils.ConfigureLogging()

	var hook = utils.NewTestLogHook()
	log.AddHook(hook)

	handleContextCancel(func() {})
	base.CancelContext()
	utils.AssertWait(t, 1, func() bool {
		return len(hook.LogEntries) > 0
	})
	assert.Equal(t, "stopping vmaas_sync", hook.LogEntries[0].Message)
}
