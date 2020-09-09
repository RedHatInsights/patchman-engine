package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var msgs []mqueue.Message

type mockKafkaWriter struct{}

func (t mockKafkaWriter) WriteMessages(_ context.Context, ev ...mqueue.Message) error {
	msgs = append(msgs, ev...)
	return nil
}

func TestSync(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	evalWriter = &mockKafkaWriter{}

	err := websocketHandler([]byte("webapps-refreshed"), nil)
	assert.Nil(t, err)

	expected := []string{"RH-100"}
	database.CheckAdvisoriesInDB(t, expected)

	evras := []string{"5.10.13-200.fc31.x86_64"}
	assert.NoError(t, database.Db.Unscoped().Where("evra in (?)", evras).Delete(&models.Package{}).Error)
	assert.NoError(t, database.Db.Unscoped().Where("name IN (?)", expected).Delete(&models.AdvisoryMetadata{}).Error)

	assert.Equal(t, 2, len(msgs))

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
	time.Sleep(time.Millisecond)
	assert.Equal(t, "stopping vmaas_sync", hook.LogEntries[0].Message)
}
