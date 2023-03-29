package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func updatesHandler(c *gin.Context) {
	data := `{
		"basearch": "i686",
		"modules_list": [],
		"releasever": "ser1",
		"repository_list": [
			"repo1"
		],
		"update_list": {
			"firefox-0:76.0.1-1.fc31.x86_64": {
				"available_updates": [
					{
						"basearch": "i686",
						"erratum": "RH-1",
						"package": "firefox-0:77.0.1-1.fc31.x86_64",
						"releasever": "ser1",
						"repository": "repo1"
					},
					{
						"basearch": "i686",
						"erratum": "RH-2",
						"package": "firefox-1:76.0.1-1.fc31.x86_64",
						"releasever": "ser1",
						"repository": "repo1"
					}
				]
			},
			"kernel-0:5.6.13-200.fc31.x86_64": {
				"available_updates": [
					{
						"basearch": "i686",
						"erratum": "RH-100",
						"package": "kernel-0:5.10.13-200.fc31.x86_64",
						"releasever": "ser1",
						"repository": "repo1"
					}
				]
			}
		}
	}`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func patchesHandler(c *gin.Context) {
	data := `{"errata_list":["RH-1","RH-2","RH-100"]}`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

// nolint: funlen
func erratasHandler(c *gin.Context) {
	data := `{
    "errata_list": {
        "RH-1": {
            "bugzilla_list": [],
            "cve_list": [],
            "description": "adv-1-des",
            "issued": "2016-09-22T12:00:00+04:00",
            "package_list": [
                "firefox-0:77.0.1-1.fc31.x86_64"
            ],
            "reference_list": [],
            "release_versions": [
                "7.0",
                "7Server"
            ],
            "requires_reboot": false,
            "solution": "adv-1-sol",
            "summary": "adv-1-sum",
            "synopsis": "adv-1-syn",
            "type": "enhancement",
            "updated": "2016-09-22T12:00:00+04:00",
            "url": "url1"
        },
        "RH-100": {
            "bugzilla_list": [],
            "cve_list": [
                "CVE-1001",
                "CVE-1002"
            ],
            "description": "adv-100-des",
            "issued": "2020-01-02T15:04:05+07:00",
            "package_list": [
                "kernel-5.10.13-200.fc31.x86_64"
            ],
            "reference_list": [],
            "requires_reboot": true,
            "solution": "adv-100-sol",
            "summary": "adv-100-sum",
            "synopsis": "adv-100-syn",
            "type": "security",
            "updated": "2020-01-02T15:04:05+07:00",
            "url": "url100"
        },
        "RH-2": {
            "bugzilla_list": [],
            "cve_list": [],
            "description": "adv-2-des",
            "issued": "2016-09-22T12:00:00+04:00",
            "package_list": [
                "firefox-1:76.0.1-1.fc31.x86_64"
            ],
            "reference_list": [],
            "requires_reboot": false,
            "solution": "adv-2-sol",
            "summary": "adv-2-sum",
            "synopsis": "adv-2-syn",
            "type": "bugfix",
            "updated": "2016-09-22T12:00:00+04:00",
            "url": "url2"
        },
        "EPEL-1234": {
            "description": "epel-des",
            "issued": "2016-09-22T12:00:00+04:00",
            "reference_list": [],
            "requires_reboot": false,
            "summary": "epel-sum",
            "synopsis": "epel-syn",
            "type": "bugfix",
            "updated": "2016-09-22T12:00:00+04:00",
            "solution": "",
            "url": ""
        }
    },
    "page": 0,
    "page_size": 10,
    "pages": 1
    }`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func pkgListHandler(c *gin.Context) {
	data := `{
    "page": 0,
    "page_size": 8,
    "pages": 1,
    "package_list": [{
			"nevra": "firefox-76.0.1-1.fc31.x86_64",
			"summary": "Mozilla Firefox Web browser",
			"description": "Mozilla Firefox is an open-source web browser..."
        },{
			"nevra": "kernel-5.6.13-200.fc31.x86_64",
			"summary": "The Linux kernel",
			"description": "The kernel meta package"		
        },{
			"nevra": "firefox-0:77.0.1-1.fc31.x86_64",
			"summary": "Mozilla Firefox Web browser",
			"description": "Mozilla Firefox is an open-source web browser..."
		},{
			"nevra": "kernel-5.7.13-200.fc31.x86_64",
			"summary": "The Linux kernel",
			"description": "The kernel meta package"
		},{
            "nevra": "firefox-0:77.0.1-1.fc31.src",
			"summary": null,
			"description": null
		},{
			"nevra": "kernel-5.7.13-200.fc31.src",
			"summary": null,
			"description": null
		},{
			"nevra": "curl-999-1.x86_64",
			"summary": "curl newest summary",
			"description": "curl newest description"
		},{
			"nevra": "bash-999-2.x86_64",
			"summary": "bash newest summary",
			"description": "bash newest description"
		}
    ],
    "last_change": "2021-04-09T04:52:06.999732+00:00"}`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func reposHandler(c *gin.Context) {
	data := `{
    "page": 0,
    "page_size": 3,
    "pages": 1,
    "repository_list": {
        "repo1": [],
        "repo2": [],
        "repo3": []
    }}`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func dbchangeHandler(c *gin.Context) {
	data := `{
        "errata_changes": "2222-04-16 20:07:58.500192+00",
        "cve_changes": "2222-04-16 20:06:47.214266+00",
        "repository_changes": "2222-04-16 20:07:55.214266+00",
        "last_change": "2222-04-16 20:07:58.500192+00",
        "exported": "2222-04-16 20:07:59.235962+00"
    }`
	c.Data(http.StatusOK, gin.MIMEJSON, []byte(data))
}

func initVMaaS(app *gin.Engine) {
	// Mock updates endpoint for VMaaS
	app.POST("/api/v3/updates", updatesHandler)
	app.POST("/api/v3/patches", patchesHandler)
	app.POST("/api/v3/errata", erratasHandler)
	app.POST("/api/v3/repos", reposHandler)
	app.POST("/api/v3/pkglist", pkgListHandler)
	app.GET("/api/v3/dbchange", dbchangeHandler)
}
