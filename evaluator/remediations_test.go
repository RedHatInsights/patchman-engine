package evaluator

import (
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testFfUpdates = []vmaas.UpdatesV2ResponseAvailableUpdates{
	{Repository: vmaas.PtrString("repo1"), Releasever: vmaas.PtrString("ser1"), Basearch: vmaas.PtrString("i686"),
		Erratum: vmaas.PtrString("RH-1"), Package: vmaas.PtrString("firefox-0:77.0.1-1.fc31.x86_64")},
	{Repository: vmaas.PtrString("repo1"), Releasever: vmaas.PtrString("ser1"), Basearch: vmaas.PtrString("i686"),
		Erratum: vmaas.PtrString("RH-2"), Package: vmaas.PtrString("firefox-1:76.0.1-1.fc31.x86_64")},
}
var testKUpdates = []vmaas.UpdatesV2ResponseAvailableUpdates{
	{Repository: vmaas.PtrString("repo1"), Releasever: vmaas.PtrString("ser1"), Basearch: vmaas.PtrString("i686"),
		Erratum: vmaas.PtrString("RH-100"), Package: vmaas.PtrString("kernel-5.10.13-200.fc31.x86_64")},
}
var testUpdateList = map[string]vmaas.UpdatesV2ResponseUpdateList{
	"firefox-0:76.0.1-1.fc31.x86_64": {
		AvailableUpdates: &testFfUpdates,
	},
	"kernel-5.6.13-200.fc31.x86_64": {
		AvailableUpdates: &testKUpdates,
	},
}
var testModuleList = []vmaas.UpdatesV3RequestModulesList{}
var testVmaasResponse = vmaas.UpdatesV2Response{
	UpdateList:     &testUpdateList,
	RepositoryList: utils.PtrSliceString([]string{"repo1"}),
	ModulesList:    &testModuleList,
	Releasever:     vmaas.PtrString("ser1"),
	Basearch:       vmaas.PtrString("i686"),
}

func TestCreateRemediationsState(t *testing.T) {
	id := "00000000-0000-0000-0000-000000000012"
	state := createRemediationsStateMsg(id, &testVmaasResponse)
	assert.NotNil(t, state)
	assert.Equal(t, state.HostID, id)
	assert.Equal(t, state.Issues, []string{"patch:RH-1", "patch:RH-100", "patch:RH-2",
		"patch:firefox-0:77.0.1-1.fc31.x86_64", "patch:firefox-1:76.0.1-1.fc31.x86_64",
		"patch:kernel-5.10.13-200.fc31.x86_64"})
}
