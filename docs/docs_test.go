package docs

import (
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

const openAPIPath = "v3/openapi.json"

func TestValidateOpenAPI3DocStr(t *testing.T) {
	doc, err := os.ReadFile(openAPIPath)
	assert.Nil(t, err)
	_, err = openapi3.NewLoader().LoadFromData(doc)
	assert.Nil(t, err)
}

func TestFilterOpenAPIPaths1(t *testing.T) {
	nRemovedPaths := filterOpenAPI(EndpointsConfig{
		EnableBaselines: true,
		EnableTemplates: true,
	}, openAPIPath, "/tmp/openapi-filter-test.json")
	assert.Equal(t, 0, nRemovedPaths)
}

func TestFilterOpenAPIPaths2(t *testing.T) {
	nRemovedPaths := filterOpenAPI(EndpointsConfig{
		EnableBaselines: false,
		EnableTemplates: false,
	}, openAPIPath, "/tmp/openapi-filter-test.json")
	assert.Equal(t, 7, nRemovedPaths)
}
