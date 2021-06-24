package vmaas_sync //nolint:revive,stylecheck
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
	database.CheckEVRAsInDB(t, 4, "77.0.1-1.fc31.src", "77.0.1-1.fc31.x86_64", // added firefox versions
		"5.7.13-200.fc31.src", "5.7.13-200.fc31.x86_64") // added kernel versions
	database.DeleteNewlyAddedPackages(t)
}
