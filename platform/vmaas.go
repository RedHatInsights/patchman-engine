package platform

import (
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

func updatesHandler(c *gin.Context) {
	data := vmaas.UpdatesV2Response{
		UpdateList: map[string]vmaas.UpdatesV2ResponseUpdateList{
			"firefox-0:76.0.1-1.fc31.x86_64": {
				AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "RH-1",
						Package:    "firefox-0:77.0.1-1.fc31.x86_64",
					},
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "RH-2",
						Package:    "firefox-1:76.0.1-1.fc31.x86_64",
					},
				},
			},
			"kernel-5.6.13-200.fc31.x86_64": {
				AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "RH-100",
						Package:    "kernel-5.10.13-200.fc31.x86_64",
					},
				},
			},
		},
		RepositoryList: []string{"repo1"},
		ModulesList:    []vmaas.UpdatesRequestModulesList{},
		Releasever:     "ser1",
		Basearch:       "i686",
	}

	c.JSON(http.StatusOK, data)
}

func patchesHandler(c *gin.Context) {
	data := vmaas.PatchesResponse{
		ErrataList: []string{
			"RH-1", "RH-2", "RH-100",
		},
	}

	c.JSON(http.StatusOK, data)
}

func erratasHandler(c *gin.Context) {
	data := vmaas.ErrataResponse{
		Page:     0,
		PageSize: 10,
		Pages:    1,
		ErrataList: map[string]vmaas.ErrataResponseErrataList{
			"RH-1": {
				Updated:       "2016-09-22T12:00:00+04:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2016-09-22T12:00:00+04:00",
				Description:   "adv-1-des",
				Solution:      "adv-1-sol",
				Summary:       "adv-1-sum",
				Url:           "url1",
				Synopsis:      "adv-1-syn",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"firefox-0:77.0.1-1.fc31.x86_64"},
				Type:          "enhancement",
			},
			"RH-2": {
				Updated:       "2016-09-22T12:00:00+04:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2016-09-22T12:00:00+04:00",
				Description:   "adv-2-des",
				Solution:      "adv-2-sol",
				Summary:       "adv-2-sum",
				Url:           "url2",
				Synopsis:      "adv-2-syn",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"firefox-1:76.0.1-1.fc31.x86_64"},
				Type:          "bugfix",
			},
			"RH-100": {
				Updated:       "2020-01-02T15:04:05+07:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2020-01-02T15:04:05+07:00",
				Description:   "adv-100-des",
				Solution:      "adv-100-sol",
				Summary:       "adv-100-sum",
				Url:           "url100",
				Synopsis:      "adv-100-syn",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"kernel-5.10.13-200.fc31.x86_64"},
				Type:          "enhancement",
			},
		},
		ModifiedSince: "",
	}
	c.JSON(http.StatusOK, data)
}

func reposHandler(c *gin.Context) {
	data := vmaas.ReposResponse{
		Page:     0,
		PageSize: 3,
		Pages:    1,
		RepositoryList: map[string][]map[string]interface{}{
			"repo1": make([]map[string]interface{}, 0),
			"repo2": make([]map[string]interface{}, 0),
			"repo3": make([]map[string]interface{}, 0),
		},
	}
	c.JSON(http.StatusOK, data)
}

var upgrader = websocket.Upgrader{} // use default options
func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("Failed to set websocket upgrade")
		return
	}
	ws := addWebsocket()
	for {
		// Wait for someone to call /control/sync
		<-ws
		// Send refresh mesage to clients
		err = conn.WriteMessage(websocket.TextMessage, []byte("webapps-refreshed"))
		if err != nil {
			panic(err)
		}
	}
}

func initVMaaS(app *gin.Engine) {
	// Mock updates endpoint for VMaaS
	app.GET("/api/v3/updates", updatesHandler)
	app.POST("/api/v3/updates", updatesHandler)
	app.GET("/api/v1/patches", patchesHandler)
	app.POST("/api/v1/patches", patchesHandler)
	// Mock erratas endpoint for VMaaS
	app.POST("/api/v1/errata", erratasHandler)
	// Mock repos endpoint for VMaaS
	app.POST("/api/v1/repos", reposHandler)
	// Mock websocket endpoint for VMaaS
	app.GET("/ws", func(context *gin.Context) {
		wshandler(context.Writer, context.Request)
	})
}
