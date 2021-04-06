package platform

import (
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

func updatesHandler(c *gin.Context) {
	ffUpdates := []vmaas.UpdatesV2ResponseAvailableUpdates{
		{
			Repository: vmaas.PtrString("repo1"),
			Releasever: vmaas.PtrString("ser1"),
			Basearch:   vmaas.PtrString("i686"),
			Erratum:    vmaas.PtrString("RH-1"),
			Package:    vmaas.PtrString("firefox-0:77.0.1-1.fc31.x86_64"),
		},
		{
			Repository: vmaas.PtrString("repo1"),
			Releasever: vmaas.PtrString("ser1"),
			Basearch:   vmaas.PtrString("i686"),
			Erratum:    vmaas.PtrString("RH-2"),
			Package:    vmaas.PtrString("firefox-1:76.0.1-1.fc31.x86_64"),
		},
	}
	kUpdates := []vmaas.UpdatesV2ResponseAvailableUpdates{
		{
			Repository: vmaas.PtrString("repo1"),
			Releasever: vmaas.PtrString("ser1"),
			Basearch:   vmaas.PtrString("i686"),
			Erratum:    vmaas.PtrString("RH-100"),
			Package:    vmaas.PtrString("kernel-5.10.13-200.fc31.x86_64"),
		},
	}
	updatesList := map[string]vmaas.UpdatesV2ResponseUpdateList{
		"firefox-0:76.0.1-1.fc31.x86_64": {AvailableUpdates: &ffUpdates},
		"kernel-5.6.13-200.fc31.x86_64":  {AvailableUpdates: &kUpdates},
	}
	moduleList := []vmaas.UpdatesV3RequestModulesList{}
	data := vmaas.UpdatesV2Response{
		UpdateList:     &updatesList,
		RepositoryList: utils.PtrSliceString([]string{"repo1"}),
		ModulesList:    &moduleList,
		Releasever:     vmaas.PtrString("ser1"),
		Basearch:       vmaas.PtrString("i686"),
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
			Updated:       vmaas.PtrString("2016-09-22T12:00:00+04:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrTimeParse("2016-09-22T12:00:00+04:00"),
			Description:   vmaas.PtrString("adv-1-des"),
			Solution:      vmaas.PtrString("adv-1-sol"),
			Summary:       vmaas.PtrString("adv-1-sum"),
			Url:           vmaas.PtrString("url1"),
			Synopsis:      vmaas.PtrString("adv-1-syn"),
			CveList:       utils.PtrSliceString([]string{}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"firefox-0:77.0.1-1.fc31.x86_64"}),
			Type:          vmaas.PtrString("enhancement"),
		},
		"RH-2": {
			Updated:       vmaas.PtrString("2016-09-22T12:00:00+04:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrTimeParse("2016-09-22T12:00:00+04:00"),
			Description:   vmaas.PtrString("adv-2-des"),
			Solution:      vmaas.PtrString("adv-2-sol"),
			Summary:       vmaas.PtrString("adv-2-sum"),
			Url:           vmaas.PtrString("url2"),
			Synopsis:      vmaas.PtrString("adv-2-syn"),
			CveList:       utils.PtrSliceString([]string{}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"firefox-1:76.0.1-1.fc31.x86_64"}),
			Type:          vmaas.PtrString("bugfix"),
		},
		"RH-100": {
			Updated:       vmaas.PtrString("2020-01-02T15:04:05+07:00"),
			Severity:      vmaas.NullableString{},
			ReferenceList: utils.PtrSliceString([]string{}),
			Issued:        utils.PtrTimeParse("2020-01-02T15:04:05+07:00"),
			Description:   vmaas.PtrString("adv-100-des"),
			Solution:      vmaas.PtrString("adv-100-sol"),
			Summary:       vmaas.PtrString("adv-100-sum"),
			Url:           vmaas.PtrString("url100"),
			Synopsis:      vmaas.PtrString("adv-100-syn"),
			CveList:       utils.PtrSliceString([]string{"CVE-1001", "CVE-1002"}),
			BugzillaList:  utils.PtrSliceString([]string{}),
			PackageList:   utils.PtrSliceString([]string{"kernel-5.10.13-200.fc31.x86_64"}),
			Type:          vmaas.PtrString("security"),
		},
	}
	modifiedSince := time.Time{}
	data := vmaas.ErrataResponse{
		Page:          vmaas.PtrFloat32(0),
		PageSize:      vmaas.PtrFloat32(10),
		Pages:         vmaas.PtrFloat32(1),
		ErrataList:    &errataList,
		ModifiedSince: &modifiedSince,
	}
	c.JSON(http.StatusOK, data)
}

func reposHandler(c *gin.Context) {
	repoList := map[string][]map[string]interface{}{
		"repo1": make([]map[string]interface{}, 0),
		"repo2": make([]map[string]interface{}, 0),
		"repo3": make([]map[string]interface{}, 0),
	}
	data := vmaas.ReposResponse{
		Page:           vmaas.PtrFloat32(0),
		PageSize:       vmaas.PtrFloat32(3),
		Pages:          vmaas.PtrFloat32(1),
		RepositoryList: &repoList,
	}
	c.JSON(http.StatusOK, data)
}

func packagesHandler(c *gin.Context) {
	packageList := map[string]vmaas.PackagesResponsePackageList{
		"firefox-0:77.0.1-1.fc31.x86_64": {
			Summary:     vmaas.PtrString("Mozilla Firefox Web browser"),
			Description: vmaas.PtrString("Mozilla Firefox is an open-source web browser...")},
		"firefox-1:76.0.1-1.fc31.x86_64": {
			Summary:     vmaas.PtrString("Mozilla Firefox Web browser"),
			Description: vmaas.PtrString("Mozilla Firefox is an open-source web browser... 2")},
		"kernel-5.6.13-200.fc31.x86_64": {
			Summary:     vmaas.PtrString("The Linux kernel"),
			Description: vmaas.PtrString("The kernel meta package")},
		"kernel-5.10.13-200.fc31.x86_64": {
			Summary:     vmaas.PtrString("The Linux kernel"),
			Description: vmaas.PtrString("The kernel meta package")},
	}
	data := vmaas.PackagesResponse{PackageList: &packageList}
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
	app.GET("/api/v3/patches", patchesHandler)
	app.POST("/api/v3/patches", patchesHandler)
	// Mock erratas endpoint for VMaaS
	app.POST("/api/v3/errata", erratasHandler)
	// Mock repos endpoint for VMaaS
	app.POST("/api/v3/repos", reposHandler)
	app.POST("/api/v3/packages", packagesHandler)
	// Mock websocket endpoint for VMaaS
	app.GET("/ws", func(context *gin.Context) {
		wshandler(context.Writer, context.Request)
	})
}
