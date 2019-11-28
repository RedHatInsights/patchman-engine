package vmaas_client

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func init() {
	mockUpdatesResp := `{"update_list": {
        "firefox": {"summary": "Mozilla...", "description": "Mozilla Firefox ...",
            "available_updates": [{"package": "firefox-2","erratum": "ER2", "repository": "repo1",
                                   "basearch": "x86_64", "releasever": "ser1"},
								  {"package": "firefox-3","erratum": "ER3", "repository": "repo1",
                                   "basearch": "x86_64", "releasever": "ser1"}]},
        "kernel": {"summary": "Kernel...", "description": "Linux kernel ...",
            "available_updates": [{"package": "kernel-2","erratum": "ER4", "repository": "repo1",
                                   "basearch": "x86_64", "releasever": "ser1"}]}
        }, "repository_list": ["repo1"], "releasever": "ser1", "basearch": "x86_64", "modules_list": []}`
	mock := VMaaSMock{UpdatesResp: mockUpdatesResp, Port: 8083}
	go mock.Run()
}

var (
	updatesUrl = "http://localhost:8083/api/v1/updates"
)

func TestCallVMaasUpdatesBasic(t *testing.T) {
	payload := `{"package_list": ["firefox-1", "kernel-1"], "repository_list": ["repo1"],
		         "modules_list": [], "releasever": "ser1", "basearch": "x86_64"}`

	resp, err := CallVMaaSUpdates(updatesUrl, payload)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, len(resp.UpdateList))
	assert.Equal(t, 2, len(resp.UpdateList["firefox"].AvailableUpdates))
	assert.Equal(t, "x86_64", resp.UpdateList["firefox"].AvailableUpdates[0].Basearch)
	assert.Equal(t, "firefox-2", resp.UpdateList["firefox"].AvailableUpdates[0].Package)
	assert.Equal(t, "ser1", resp.UpdateList["firefox"].AvailableUpdates[0].Releasever)
	assert.Equal(t, "repo1", resp.UpdateList["firefox"].AvailableUpdates[0].Repository)
	assert.Equal(t, "ER2", resp.UpdateList["firefox"].AvailableUpdates[0].Erratum)
	assert.Equal(t, 1, len(resp.UpdateList["kernel"].AvailableUpdates))
	assert.Equal(t, "ser1", resp.Releasever)
	assert.Equal(t, "x86_64", resp.Basearch)
}

func TestCallVMaasUpdatesBadReq(t *testing.T) {
	payload := `{"package_list": invalid}`
	resp, err := CallVMaaSUpdates(updatesUrl, payload)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.NotNil(t, resp)
}
