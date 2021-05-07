package platform

import (
	"app/base/utils"
	"net/http"

	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func updatesHandler(c *gin.Context) {
	updates1 := []vmaas.UpdatesV2ResponseAvailableUpdates{
		{
			Repository: utils.PtrString("repo1"),
			Releasever: utils.PtrString("ser1"),
			Basearch:   utils.PtrString("i686"),
			Erratum:    utils.PtrString("RH-1"),
			Package:    utils.PtrString("firefox-77.0.1-1.fc31.x86_64"),
		},
		{
			Repository: utils.PtrString("repo1"),
			Releasever: utils.PtrString("ser1"),
			Basearch:   utils.PtrString("i686"),
			Erratum:    utils.PtrString("RH-2"),
			Package:    utils.PtrString("firefox-1:76.0.1-1.fc31.x86_64"),
		},
	}
	updates2 := []vmaas.UpdatesV2ResponseAvailableUpdates{
		{
			Repository: utils.PtrString("repo1"),
			Releasever: utils.PtrString("ser1"),
			Basearch:   utils.PtrString("i686"),
			Erratum:    utils.PtrString("RH-100"),
			Package:    utils.PtrString("kernel-5.10.13-200.fc31.x86_64"),
		},
	}
	updatesList := map[string]vmaas.UpdatesV2ResponseUpdateList{
		"firefox-76.0.1-1.fc31.x86_64":  {AvailableUpdates: &updates1},
		"kernel-5.6.13-200.fc31.x86_64": {AvailableUpdates: &updates2},
	}
	moduleList := []vmaas.UpdatesV3RequestModulesList{}
	data := vmaas.UpdatesV2Response{
		UpdateList:     &updatesList,
		RepositoryList: utils.PtrSliceString([]string{"repo1"}),
		ModulesList:    &moduleList,
		Releasever:     utils.PtrString("ser1"),
		Basearch:       utils.PtrString("i686"),
	}

	c.JSON(http.StatusOK, data)
}

func patchesHandler(c *gin.Context) {
	data := vmaas.PatchesResponse{
		ErrataList: utils.PtrSliceString([]string{
			"RH-1", "RH-2", "RH-100",
		}),
	}

	c.JSON(http.StatusOK, data)
}

func erratasHandler(c *gin.Context) {
	errataList := map[string]vmaas.ErrataResponseErrataList{
		"RH-1": {
			Updated:       utils.PtrString("2016-09-22T12:00:00+04:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrString("2016-09-22T12:00:00+04:00"),
			Description:   utils.PtrString("adv-1-des"),
			Solution:      utils.PtrString("adv-1-sol"),
			Summary:       utils.PtrString("adv-1-sum"),
			Url:           utils.PtrString("url1"),
			Synopsis:      utils.PtrString("adv-1-syn"),
			CveList:       utils.PtrSliceString([]string{}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"firefox-0:77.0.1-1.fc31.x86_64"}),
			Type:          utils.PtrString("enhancement"),
		},
		"RH-2": {
			Updated:       utils.PtrString("2016-09-22T12:00:00+04:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrString("2016-09-22T12:00:00+04:00"),
			Description:   utils.PtrString("adv-2-des"),
			Solution:      utils.PtrString("adv-2-sol"),
			Summary:       utils.PtrString("adv-2-sum"),
			Url:           utils.PtrString("url2"),
			Synopsis:      utils.PtrString("adv-2-syn"),
			CveList:       utils.PtrSliceString([]string{}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"firefox-1:76.0.1-1.fc31.x86_64"}),
			Type:          utils.PtrString("bugfix"),
		},
		"RH-100": {
			Updated:       utils.PtrString("2020-01-02T15:04:05+07:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrString("2020-01-02T15:04:05+07:00"),
			Description:   utils.PtrString("adv-100-des"),
			Solution:      utils.PtrString("adv-100-sol"),
			Summary:       utils.PtrString("adv-100-sum"),
			Url:           utils.PtrString("url100"),
			Synopsis:      utils.PtrString("adv-100-syn"),
			CveList:       utils.PtrSliceString([]string{"CVE-1001", "CVE-1002"}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"kernel-5.10.13-200.fc31.x86_64"}),
			Type:          utils.PtrString("security"),
		},
	}
	data := vmaas.ErrataResponse{
		Page:          utils.PtrFloat32(0),
		PageSize:      utils.PtrFloat32(10),
		Pages:         utils.PtrFloat32(1),
		ErrataList:    &errataList,
		ModifiedSince: utils.PtrString(""),
	}
	c.JSON(http.StatusOK, data)
}

func pkgTreeHandler(c *gin.Context) {
	data := `{
    "page": 0,
    "page_size": 3,
    "pages": 1,
    "package_name_list": {
        "firefox": [{
                "nevra": "firefox-76.0.1-1.fc31.x86_64",
                "summary": "Mozilla Firefox Web browser",
                "description": "Mozilla Firefox is an open-source web browser..."
            },{
                "nevra": "firefox-0:77.0.1-1.fc31.x86_64",
                "summary": "Mozilla Firefox Web browser",
                "description": "Mozilla Firefox is an open-source web browser..."
            },{
                "nevra": "firefox-0:77.0.1-1.fc31.src",
                "summary": null,
                "description": null}
        ],
        "kernel": [{
                "nevra": "kernel-5.6.13-200.fc31.x86_64",
                "summary": "The Linux kernel",
                "description": "The kernel meta package"
            },{
                "nevra": "kernel-5.7.13-200.fc31.x86_64",
                "summary": "The Linux kernel",
                "description": "The kernel meta package"
            },{
                "nevra": "kernel-5.7.13-200.fc31.src",
                "summary": null,
                "description": null
            }
        ]},
    "last_change": "2021-04-09T04:52:06.999732+00:00"}`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func reposHandler(c *gin.Context) {
	repoList := map[string][]map[string]interface{}{
		"repo1": make([]map[string]interface{}, 0),
		"repo2": make([]map[string]interface{}, 0),
		"repo3": make([]map[string]interface{}, 0),
	}
	data := vmaas.ReposResponse{
		Page:           utils.PtrFloat32(0),
		PageSize:       utils.PtrFloat32(3),
		Pages:          utils.PtrFloat32(1),
		RepositoryList: &repoList,
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
	app.POST("/api/v3/updates", updatesHandler)
	app.POST("/api/v3/patches", patchesHandler)
	app.POST("/api/v3/errata", erratasHandler)
	app.POST("/api/v3/repos", reposHandler)
	app.POST("/api/v3/pkgtree", pkgTreeHandler)
	// Mock websocket endpoint for VMaaS
	app.GET("/ws", func(context *gin.Context) {
		wshandler(context.Writer, context.Request)
	})
}
