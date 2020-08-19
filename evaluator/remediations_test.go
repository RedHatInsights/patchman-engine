package evaluator

import (
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testVmaasResponse = vmaas.UpdatesV2Response{
	UpdateList: map[string]vmaas.UpdatesV2ResponseUpdateList{
		"firefox-0:76.0.1-1.fc31.x86_64": {
			AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
				{Repository: "repo1", Releasever: "ser1", Basearch: "i686", Erratum: "RH-1",
					Package: "firefox-0:77.0.1-1.fc31.x86_64"},
				{Repository: "repo1", Releasever: "ser1", Basearch: "i686", Erratum: "RH-2",
					Package: "firefox-1:76.0.1-1.fc31.x86_64"},
			},
		},
		"kernel-5.6.13-200.fc31.x86_64": {
			AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
				{Repository: "repo1", Releasever: "ser1", Basearch: "i686", Erratum: "RH-100",
					Package: "kernel-5.10.13-200.fc31.x86_64"},
			},
		},
	},
	RepositoryList: []string{"repo1"},
	ModulesList:    nil,
	Releasever:     "ser1",
	Basearch:       "i686",
}

func TestCreateRemediationsState(t *testing.T) {
	id := "00000000-0000-0000-0000-000000000012"
	state := createRemediationsStateMsg(id, testVmaasResponse)
	assert.NotNil(t, state)
	assert.Equal(t, state.HostID, id)
	assert.Equal(t, state.Issues, []string{"patch:RH-1", "patch:RH-100", "patch:RH-2",
		"patch:firefox-0:77.0.1-1.fc31.x86_64", "patch:firefox-1:76.0.1-1.fc31.x86_64",
		"patch:kernel-5.10.13-200.fc31.x86_64"})
}
