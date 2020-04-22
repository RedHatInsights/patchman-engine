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
			"firefox": {
				AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "ER1",
						Package:    "firefox-2",
					},
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "ER2",
						Package:    "firefox-3",
					},
				},
			},
			"kernel": {
				AvailableUpdates: []vmaas.UpdatesResponseAvailableUpdates{
					{
						Repository: "repo1",
						Releasever: "ser1",
						Basearch:   "i686",
						Erratum:    "ER3",
						Package:    "kernel-2",
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
			"ER1", "ER2", "ER3",
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
			"ER1": {
				Updated:       "2006-01-02T15:04:05+07:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2006-01-02T15:04:05+07:00",
				Description:   "Simple Errata",
				Solution:      "Do nothing",
				Summary:       "Simple errata",
				Url:           "http://google.com",
				Synopsis:      "Data",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"firefox-2.ser1.i686"},
				Type:          "enhancement",
			},
			"ER2": {
				Updated:       "2006-01-02T15:04:05+07:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2006-01-02T15:04:05+07:00",
				Description:   "Simple Errata",
				Solution:      "Do nothing",
				Summary:       "Simple errata",
				Url:           "http://google.com",
				Synopsis:      "Data",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"firefox-3.ser1.i686"},
				Type:          "enhancement",
			},
			"ER3": {
				Updated:       "2006-01-02T15:04:05+07:00",
				Severity:      nil,
				ReferenceList: []string{},
				Issued:        "2006-01-02T15:04:05+07:00",
				Description:   "Simple Errata",
				Solution:      "Do nothing",
				Summary:       "Simple errata",
				Url:           "http://google.com",
				Synopsis:      "Data",
				CveList:       []string{},
				BugzillaList:  []string{},
				PackageList:   []string{"kernel-2.ser1.i686"},
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
