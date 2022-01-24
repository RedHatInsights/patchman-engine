package docs

import (
	"app/base/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"strings"
)

const origOpenapiPath = "./docs/openapi.json"
const exposedOpenapiPath = "/tmp/openapi.json"
const OpenapiURL = "/api/patch/v1/openapi.json"

type EndpointsConfig struct {
	EnableBaselines bool
}

func Init(app *gin.Engine, config EndpointsConfig) {
	nRemovedPaths := filterOpenAPI(config, origOpenapiPath, exposedOpenapiPath)
	utils.Log("nRemovedPaths", nRemovedPaths).Debug("Filtering endpoints paths from openapi.json")
	app.GET(OpenapiURL, handleOpenapiSpec)
}

func handleOpenapiSpec(c *gin.Context) {
	c.Status(200)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.File(exposedOpenapiPath)
}

func filterOpenAPI(config EndpointsConfig, inputOpenapiPath, outputOpenapiPath string) (removedPaths int) {
	doc, err := ioutil.ReadFile(inputOpenapiPath)
	panicErr(err)

	sw, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(doc)
	panicErr(err)

	filteredPaths := openapi3.Paths{}
	for path := range sw.Paths {
		if !config.EnableBaselines && strings.Contains(path, "v1/baselines") {
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
