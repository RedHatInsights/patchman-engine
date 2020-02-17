package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

var events []mqueue.PlatformEvent

type mockKafkaWriter struct{}

func (t mockKafkaWriter) WriteEvent(_ context.Context, ev mqueue.PlatformEvent) error {
	events = append(events, ev)
	return nil
}

func TestSync(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	evalWriter = &mockKafkaWriter{}

	err := websocketHandler([]byte("webapps-refreshed"), nil)
	assert.Nil(t, err)

	expected := []string{"ER1", "ER2", "ER3"}
	database.CheckAdvisoriesInDb(t, expected)
	assert.Nil(t, database.Db.Unscoped().Where("name IN (?)", expected).Delete(&models.AdvisoryMetadata{}).Error)

	assert.Equal(t, 12, len(events))
}
