package listener

import (
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestVMaaSGetUpdates(t *testing.T) {
	utils.SkipWithoutPlatform(t)

	configure()

	vmaasCallArgs := vmaas.AppUpdatesHandlerV2PostPostOpts{}
	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV2PostPost(context.Background(), &vmaasCallArgs)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, len(vmaasData.UpdateList["firefox"].AvailableUpdates))
	assert.Equal(t, 1, len(vmaasData.UpdateList["kernel"].AvailableUpdates))
}
