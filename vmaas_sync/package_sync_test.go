package vmaas_sync //nolint:golint,stylecheck
import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSyncPackages(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	err := syncPackages(time.Now(), nil)
	assert.NoError(t, err)

	database.CheckPackagesNamesInDB(t, "bash", "curl")
	database.DeleteNewlyAddedPackages(t)
}
