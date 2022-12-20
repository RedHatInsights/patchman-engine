package docs

import (
	"app/base/utils"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
)

const exposedOpenapiPathV1 = "/tmp/openapi.v1.json"
const exposedOpenapiPathV2 = "/tmp/openapi.v2.json"
const exposedOpenapiPathAdmin = "/tmp/openapi.admin.json"

var appVersions = map[string]openapiData{
	"v1": {
		in: "./docs/v1/openapi.json", out: exposedOpenapiPathV1,
		url: "/api/patch/v1/openapi.json", handler: handleOpenapiV1Spec,
	},
	"v2": {
		in: "./docs/v2/openapi.json", out: exposedOpenapiPathV2,
		url: "/api/patch/v2/openapi.json", handler: handleOpenapiV2Spec,
	},
}
var adminAPI = openapiData{
	in: "./docs/admin/openapi.json", out: exposedOpenapiPathAdmin,
	url: "/api/patch/admin/openapi.json", handler: handleOpenapiAdminSpec,
}

type openapiData struct {
	in      string
	out     string
	url     string
	handler func(*gin.Context)
}

type EndpointsConfig struct {
	EnableBaselines bool
}

func Init(app *gin.Engine, config EndpointsConfig) string {
	var ver string
	var data openapiData
	for ver, data = range appVersions {
		nRemovedPaths := filterOpenAPI(config, data.in, data.out)
		utils.Log("nRemovedPaths", nRemovedPaths).Debug("Filtering endpoints paths from " + ver + "/openapi.json")
		app.GET(data.url, data.handler)
	}

	return data.url
}

func InitAdminAPI(app *gin.Engine) string {
	cfg := EndpointsConfig{}
	// used to create file with openapi.json
	filterOpenAPI(cfg, adminAPI.in, adminAPI.out)
	app.GET(adminAPI.url, adminAPI.handler)
	return adminAPI.url
}

func handleOpenapiV1Spec(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File(exposedOpenapiPathV1)
}

func handleOpenapiV2Spec(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File(exposedOpenapiPathV2)
}

func handleOpenapiAdminSpec(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File(exposedOpenapiPathAdmin)
}

func filterOpenAPI(config EndpointsConfig, inputOpenapiPath, outputOpenapiPath string) (removedPaths int) {
	doc, err := ioutil.ReadFile(inputOpenapiPath)
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

	err = ioutil.WriteFile(outputOpenapiPath, outputBytes, 0600)
	panicErr(err)
	return removedPaths
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
