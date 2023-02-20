package docs

import (
	"app/base/utils"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
)

const exposedOpenapiPathV1 = "/tmp/openapi.v1.json"
const exposedOpenapiPathV2 = "/tmp/openapi.v2.json"
const exposedOpenapiPathV3 = "/tmp/openapi.v3.json"
const exposedOpenapiPathAdmin = "/tmp/openapi.admin.json"

var appVersions = map[int]openapiData{
	1: {
		in: "./docs/v1/openapi.json", out: exposedOpenapiPathV1,
		url: "/api/patch/v1/openapi.json",
	},
	2: {
		in: "./docs/v2/openapi.json", out: exposedOpenapiPathV2,
		url: "/api/patch/v2/openapi.json",
	},
	3: {
		in: "./docs/v3/openapi.json", out: exposedOpenapiPathV3,
		url: "/api/patch/v3/openapi.json",
	},
}
var adminAPI = openapiData{
	in: "./docs/admin/openapi.json", out: exposedOpenapiPathAdmin,
	url: "/api/patch/admin/openapi.json",
}

type openapiData struct {
	in  string
	out string
	url string
}

type EndpointsConfig struct {
	EnableBaselines bool
}

func Init(app *gin.Engine, config EndpointsConfig) string {
	maxVer := 1
	for ver, data := range appVersions {
		if ver > maxVer {
			maxVer = ver
		}
		nRemovedPaths := filterOpenAPI(config, data.in, data.out)
		utils.LogDebug("nRemovedPaths", nRemovedPaths, fmt.Sprintf("Filtering endpoints paths from %d/openapi.json", ver))
		app.GET(data.url, getOpenapiHandler(ver))
	}

	return appVersions[maxVer].url
}

func InitAdminAPI(app *gin.Engine) string {
	cfg := EndpointsConfig{}
	// used to create file with openapi.json
	filterOpenAPI(cfg, adminAPI.in, adminAPI.out)
	app.GET(adminAPI.url, handleOpenapiAdminSpec)
	return adminAPI.url
}

func getOpenapiHandler(ver int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusOK)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.File(appVersions[ver].out)
	}
}

func handleOpenapiAdminSpec(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File(exposedOpenapiPathAdmin)
}

func filterOpenAPI(config EndpointsConfig, inputOpenapiPath, outputOpenapiPath string) (removedPaths int) {
	doc, err := os.ReadFile(inputOpenapiPath)
	panicErr(err)

	sw, err := openapi3.NewLoader().LoadFromData(doc)
	panicErr(err)

	filteredPaths := openapi3.Paths{}
	for path := range sw.Paths {
		if !config.EnableBaselines && strings.Contains(path, "/baselines") {
			removedPaths++
			continue
		}
		filteredPaths[path] = sw.Paths[path]
	}

	sw.Paths = filteredPaths
	outputBytes, err := sw.MarshalJSON()
	panicErr(err)

	err = os.WriteFile(outputOpenapiPath, outputBytes, 0600)
	panicErr(err)
	return removedPaths
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
